package buffer

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ingestBufState int

const (
	started ingestBufState = iota
	initialized
	finished
)

func (s ingestBufState) String() string {
	return [...]string{"started", "initialized", "finished"}[s]
}

// getGID returns the current goroutine ID
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// e.g. T is eventOpts and U is *dbsqlc.Event

type IngestBuf[T any, U any] struct {
	name       string // a human readable name for the buffer
	outputFunc func(ctx context.Context, items []T) ([]U, error)
	sizeFunc   func(T) int

	state       ingestBufState
	maxCapacity int           // max number of items to hold in buffer before we flush
	flushPeriod time.Duration // max time to hold items in buffer before we flush

	inputChan          chan *inputWrapper[T, U]
	lastFlush          time.Time
	internalArr        []*inputWrapper[T, U]
	sizeOfData         int // size of data in buffer
	maxDataSizeInQueue int // max number of bytes to hold in buffer before we flush

	l                 *zerolog.Logger
	lock              sync.Mutex
	ctx               context.Context
	cancel            context.CancelFunc
	flushSemaphore    *semaphore.Weighted
	waitForFlush      time.Duration
	maxConcurrent     int
	fmux              sync.Mutex
	currentlyFlushing int
	debugMap          sync.Map
}

type inputWrapper[T any, U any] struct {
	item     T
	doneChan chan<- *FlushResponse[U]
}

type IngestBufOpts[T any, U any] struct {
	Name string `validate:"required"`
	// MaxCapacity is the maximum number of items to hold in buffer before we initiate a flush
	MaxCapacity        int                                               `validate:"required,gt=0"`
	FlushPeriod        time.Duration                                     `validate:"required,gt=0"`
	MaxDataSizeInQueue int                                               `validate:"required,gt=0"`
	OutputFunc         func(ctx context.Context, items []T) ([]U, error) `validate:"required"`
	SizeFunc           func(T) int                                       `validate:"required"`
	L                  *zerolog.Logger                                   `validate:"required"`
	MaxConcurrent      int                                               `validate:"omitempty,gt=0"`
	WaitForFlush       time.Duration                                     `validate:"omitempty,gt=0"`
}

// NewIngestBuffer creates a new buffer for any type T
func NewIngestBuffer[T any, U any](opts IngestBufOpts[T, U]) *IngestBuf[T, U] {

	inputChannelSize := opts.MaxCapacity
	if inputChannelSize < 100 {
		inputChannelSize = 100
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	logger := opts.L.With().Str("buffer", opts.Name).Logger()
	if opts.MaxConcurrent == 0 {
		opts.MaxConcurrent = 50
	}

	if opts.WaitForFlush == 0 {
		opts.WaitForFlush = 1 * time.Millisecond

	}

	return &IngestBuf[T, U]{
		name:               opts.Name,
		state:              initialized,
		maxCapacity:        opts.MaxCapacity,
		flushPeriod:        opts.FlushPeriod,
		inputChan:          make(chan *inputWrapper[T, U], inputChannelSize),
		lastFlush:          time.Now(),
		internalArr:        make([]*inputWrapper[T, U], 0),
		sizeOfData:         0,
		maxDataSizeInQueue: opts.MaxDataSizeInQueue,
		outputFunc:         opts.OutputFunc,
		sizeFunc:           opts.SizeFunc,
		l:                  &logger,
		ctx:                ctx,
		cancel:             cancel,
		flushSemaphore:     semaphore.NewWeighted(int64(opts.MaxConcurrent)),
		waitForFlush:       opts.WaitForFlush,
		maxConcurrent:      opts.MaxConcurrent,
		currentlyFlushing:  0,
	}
}

func (b *IngestBuf[T, U]) safeAppendInternalArray(e *inputWrapper[T, U]) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.internalArr = append(b.internalArr, e)
}

func (b *IngestBuf[T, U]) safeFetchSizeOfData() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.sizeOfData
}

func (b *IngestBuf[T, U]) safeIncSizeOfData(size int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sizeOfData += size

}

func (b *IngestBuf[T, U]) safeDecSizeOfData(size int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sizeOfData -= size

}

func (b *IngestBuf[T, U]) safeSetLastFlush(t time.Time) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.lastFlush = t

}

func (b *IngestBuf[T, U]) safeFetchLastFlush() time.Time {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.lastFlush
}

// We take items from the input channel and add them to the internal array
// We then check if we need to flush the buffer and if we do we flush
// When we can't acquire a semaphore (we are over MaxConcurrent) we wait a small amount of time (waitForFlush) and then come back and add more items from the input channel to the internal array
// The input channel is buffered and can accept up to maxCapacity items (or 100 if maxCapacity is less than 100)
// If we are not emptying the buffer fast enough Buffering an item using BuffItem will eventually block and if it is blocked for long enough will error out with Resource Exhausted
// If the internal buffer array ever gets to 50X maxCapacity we will error out with Resource Exhausted

func (b *IngestBuf[T, U]) buffWorker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case e := <-b.inputChan:
			b.safeAppendInternalArray(e)
			b.safeIncSizeOfData(b.calcSizeOfData([]T{e.item}))

			if b.safeCheckSizeOfBuffer() >= b.maxCapacity || b.safeFetchSizeOfData() >= b.maxDataSizeInQueue {
				b.flush()
			}
		case <-time.After(time.Until(b.safeFetchLastFlush().Add(b.flushPeriod))):

			b.flush()

		}
	}
}

func (b *IngestBuf[T, U]) sliceInternalArray() (items []*inputWrapper[T, U]) {

	if b.safeCheckSizeOfBuffer() >= b.maxCapacity {
		b.lock.Lock()
		defer b.lock.Unlock()
		items = b.internalArr[:b.maxCapacity]
		b.internalArr = b.internalArr[b.maxCapacity:]
	} else {
		b.lock.Lock()
		defer b.lock.Unlock()
		items = b.internalArr
		b.internalArr = nil
	}
	return items
}

type FlushResponse[U any] struct {
	Result U
	Err    error
}

func (b *IngestBuf[T, U]) calcSizeOfData(items []T) int {
	size := 0
	for _, item := range items {
		size += b.sizeFunc(item)
	}
	return size
}

func (b *IngestBuf[T, U]) safeCheckSizeOfBuffer() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	return len(b.internalArr)
}

func (b *IngestBuf[T, U]) flush() {

	// need to set this before we acquire the semaphore so that we don't spin
	b.safeSetLastFlush(time.Now())

	// wait for a waitForFlush amount to acquire a semaphore
	sCtx, _ := context.WithTimeoutCause(context.Background(), b.waitForFlush, fmt.Errorf("timed out waiting for semaphore in flush"))

	err := b.flushSemaphore.Acquire(sCtx, 1)

	if err != nil {
		b.l.Warn().Msg(b.debugBuffer())
		b.l.Warn().Msgf("could not acquire semaphore in: %s  %v", b.waitForFlush, err)
		return
	}

	items := b.sliceInternalArray()
	numItems := len(items)
	if numItems == 0 {
		b.safeSetLastFlush(time.Now())
		// nothing to flush
		b.flushSemaphore.Release(1)

		return
	}

	var doneChans []chan<- *FlushResponse[U]
	opts := make([]T, numItems)

	for i := 0; i < numItems; i++ {
		opts[i] = items[i].item
		doneChans = append(doneChans, items[i].doneChan)
	}

	b.safeDecSizeOfData(b.calcSizeOfData(opts))

	go func() {

		b.fmux.Lock()
		b.currentlyFlushing++
		b.fmux.Unlock()

		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("[%s] panic recovered in flush: %v", b.name, r)
				b.l.Error().Msgf("Panic recovered: %v. Stack %s", err, string(debug.Stack()))

				// Send error to all done channels
				for _, doneChan := range doneChans {
					select {
					case doneChan <- &FlushResponse[U]{Err: err}:
					default:
						b.l.Error().Msgf("could not send panic error to done chan: %v", err)
					}
				}
			}
		}()

		defer func() {
			b.fmux.Lock()
			b.currentlyFlushing--
			b.fmux.Unlock()
		}()
		defer b.flushSemaphore.Release(1)
		// get goroutine id
		goRoutineID := fmt.Sprintf("%d", getGID())

		b.debugMap.Store(goRoutineID, fmt.Sprintf("flushing %d items at %s", len(items), time.Now().Format("2006-01-02T15:04:05.000000Z07:00")))
		defer b.debugMap.Delete(goRoutineID)

		ctx := context.Background()
		result, err := b.outputFunc(ctx, opts)

		if err != nil {
			for _, doneChan := range doneChans {
				select {
				case doneChan <- &FlushResponse[U]{Err: err}:
				default:
					b.l.Error().Msgf("could not send error to done chan: %v", err)
				}
			}
			return
		}

		for i, d := range doneChans {
			select {
			case d <- &FlushResponse[U]{Result: result[i], Err: nil}:
			default:
				b.l.Error().Msg("could not send done to done chan")
			}
		}

		b.l.Debug().Msgf("flushed %d items", numItems)
	}()
}

func (b *IngestBuf[T, U]) cleanup() error {

	b.lock.Lock()
	b.state = finished
	b.lock.Unlock()

	g := errgroup.Group{}
	g.SetLimit(b.maxConcurrent)

	for b.safeCheckSizeOfBuffer() > 0 {
		g.Go(func() error {
			b.flush()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// wait until currentlyFlushing is 0

	for {
		b.fmux.Lock()
		flushingCount := b.currentlyFlushing
		b.fmux.Unlock()
		if flushingCount == 0 {
			break
		}
		b.l.Info().Msgf("cleanup: waiting for %d goroutines to finish flushing", flushingCount)
		time.Sleep(100 * time.Millisecond)
	}

	b.cancel()

	return nil
}

func (b *IngestBuf[T, U]) Start() (func() error, error) {
	b.l.Debug().Msg("Starting buffer")

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.state == started {
		return nil, fmt.Errorf("buffer already started")
	}
	b.state = started

	go b.buffWorker()

	return b.cleanup, nil
}

func (b *IngestBuf[T, U]) StartDebugLoop() {
	b.l.Debug().Msg("starting debug loop")
	for {
		select {
		case <-time.After(10 * time.Second):
			b.l.Debug().Msg(b.debugBuffer())
		case <-b.ctx.Done():
			b.l.Debug().Msg("stopping debug loop")
			return
		}
	}
}

func (b *IngestBuf[T, U]) BuffItem(item T) (chan *FlushResponse[U], error) {

	if b.state != started {
		return nil, fmt.Errorf("buffer not ready, in state '%v'", b.state.String())
	}

	if b.safeCheckSizeOfBuffer() >= b.maxCapacity*50 {
		return nil, status.Errorf(codes.ResourceExhausted, "buffer is out of space %v", b.safeCheckSizeOfBuffer())
	}

	if b.safeCheckSizeOfBuffer() > b.maxCapacity*10 && b.safeCheckSizeOfBuffer()%1000 == 0 {
		b.l.Warn().Msgf("buffer is backed up with %d items", b.safeCheckSizeOfBuffer())
	}

	doneChan := make(chan *FlushResponse[U], 1)

	select {
	case b.inputChan <- &inputWrapper[T, U]{
		item:     item,
		doneChan: doneChan,
	}:
	case <-time.After(5 * time.Second):
		return nil, status.Errorf(codes.ResourceExhausted, "timeout waiting for buffer")

	case <-b.ctx.Done():
		return nil, fmt.Errorf("buffer is closed")
	}
	return doneChan, nil
}

func (b *IngestBuf[T, U]) countDebugMapEntries() int {
	count := 0
	b.debugMap.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
func (b *IngestBuf[T, U]) debugBuffer() string {
	var builder strings.Builder
	b.fmux.Lock()
	defer b.fmux.Unlock()

	builder.WriteString("============= Buffer =============\n")
	builder.WriteString(fmt.Sprintf("%d items in buffer\n", b.safeCheckSizeOfBuffer()))
	builder.WriteString(fmt.Sprintf("The following %d goroutines are flushing\n", b.countDebugMapEntries()))
	builder.WriteString(fmt.Sprintf("Last flushed at %v\n", b.safeFetchLastFlush()))
	builder.WriteString(fmt.Sprintf("%d max capacity\n", b.maxCapacity))
	builder.WriteString(fmt.Sprintf("%d max data size in queue\n", b.maxDataSizeInQueue))
	builder.WriteString(fmt.Sprintf("%v flush period\n", b.flushPeriod))
	builder.WriteString(fmt.Sprintf("%v wait for flush\n", b.waitForFlush))
	builder.WriteString(fmt.Sprintf("%d max concurrent\n", b.maxConcurrent))
	builder.WriteString(fmt.Sprintf("In state %v\n", b.state))
	builder.WriteString(fmt.Sprintf("%d currently flushing\n", b.currentlyFlushing))
	builder.WriteString(fmt.Sprintf("The following %d goroutines are flushing\n", b.countDebugMapEntries()))

	b.debugMap.Range(func(key, value interface{}) bool {
		builder.WriteString(fmt.Sprintf("%s %s\n", key, value))
		return true
	})

	builder.WriteString("=====================================\n")

	return builder.String()
}
