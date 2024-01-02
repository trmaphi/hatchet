package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations"
	"github.com/rs/zerolog"
)

type actionFunc func(args ...any) []any

// Action is an individual action that can be run by the worker.
type Action interface {
	// Name returns the name of the action
	Name() string

	// Run runs the action
	Run(args ...any) []any

	MethodFn() any
}

type actionImpl struct {
	name   string
	run    actionFunc
	method any
}

func (j *actionImpl) Name() string {
	return j.name
}

func (j *actionImpl) Run(args ...interface{}) []interface{} {
	return j.run(args...)
}

func (j *actionImpl) MethodFn() any {
	return j.method
}

type Worker struct {
	client client.Client

	name string

	actions map[string]Action

	l *zerolog.Logger

	cancelMap sync.Map

	services sync.Map

	alerter errors.Alerter
}

type WorkerOpt func(*WorkerOpts)

type WorkerOpts struct {
	client client.Client
	name   string
	l      *zerolog.Logger

	integrations []integrations.Integration
	alerter      errors.Alerter
}

func defaultWorkerOpts() *WorkerOpts {
	logger := zerolog.New(os.Stdout)

	return &WorkerOpts{
		name:         getHostName(),
		l:            &logger,
		integrations: []integrations.Integration{},
		alerter:      errors.NoOpAlerter{},
	}
}

func WithName(name string) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.name = name
	}
}

func WithClient(client client.Client) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.client = client
	}
}

func WithIntegration(integration integrations.Integration) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.integrations = append(opts.integrations, integration)
	}
}

func WithErrorAlerter(alerter errors.Alerter) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.alerter = alerter
	}
}

// NewWorker creates a new worker instance
func NewWorker(fs ...WorkerOpt) (*Worker, error) {
	opts := defaultWorkerOpts()

	for _, f := range fs {
		f(opts)
	}

	w := &Worker{
		client:  opts.client,
		name:    opts.name,
		l:       opts.l,
		actions: map[string]Action{},
		alerter: opts.alerter,
	}

	// register all integrations
	for _, integration := range opts.integrations {
		actions := integration.Actions()
		integrationId := integration.GetId()

		for _, integrationAction := range actions {
			action := fmt.Sprintf("%s:%s", integrationId, integrationAction)

			err := w.registerAction(action, integration.ActionHandler(integrationAction))

			if err != nil {
				return nil, fmt.Errorf("could not register integration action %s: %w", action, err)
			}
		}
	}

	w.NewService("default")

	return w, nil
}

func (w *Worker) NewService(name string) *Service {
	svc := &Service{
		Name:   name,
		worker: w,
	}

	w.services.Store(name, svc)

	return svc
}

func (w *Worker) On(t triggerConverter, workflow workflowConverter) error {
	// get the default service
	svc, ok := w.services.Load("default")

	if !ok {
		return fmt.Errorf("could not load default service")
	}

	return svc.(*Service).On(t, workflow)
}

func (w *Worker) RegisterAction(name string, method any) error {
	// get the default service
	svc, ok := w.services.Load("default")

	if !ok {
		return fmt.Errorf("could not load default service")
	}

	return svc.(*Service).RegisterAction(method)
}

func (w *Worker) registerAction(name string, method any) error {
	actionFunc, err := getFnFromMethod(method)

	if err != nil {
		return fmt.Errorf("could not get function from method: %w", err)
	}

	// ensure action has not been registered
	if _, ok := w.actions[name]; ok {
		return fmt.Errorf("action %s already registered", name)
	}

	w.actions[name] = &actionImpl{
		name:   name,
		run:    actionFunc,
		method: method,
	}

	return nil
}

// Start starts the worker in blocking fashion
func (w *Worker) Start(ctx context.Context) error {
	actionNames := []string{}

	for _, job := range w.actions {
		actionNames = append(actionNames, job.Name())
	}

	listener, err := w.client.Dispatcher().GetActionListener(ctx, &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    actionNames,
	})

	if err != nil {
		return fmt.Errorf("could not get action listener: %w", err)
	}

	errCh := make(chan error)

	actionCh, err := listener.Actions(ctx, errCh)

	if err != nil {
		return fmt.Errorf("could not get action channel: %w", err)
	}

RunWorker:
	for {
		select {
		case err := <-errCh:
			w.l.Err(err).Msg("action listener error")
			break RunWorker
		case action := <-actionCh:
			go func(action *client.Action) {
				res, err := w.executeAction(context.Background(), action)

				if err != nil {
					w.l.Error().Err(err).Msgf("could not execute action: %s", action.ActionId)
				}

				w.l.Debug().Msgf("action %s completed with result %v", action.ActionId, res)
			}(action)
		case <-ctx.Done():
			w.l.Debug().Msgf("worker %s received context done, stopping", w.name)
			break RunWorker
		}
	}

	w.l.Debug().Msgf("worker %s stopped", w.name)

	err = listener.Unregister()

	if err != nil {
		return fmt.Errorf("could not unregister worker: %w", err)
	}

	return nil
}

func (w *Worker) executeAction(ctx context.Context, assignedAction *client.Action) (result any, err error) {
	if assignedAction.ActionType == client.ActionTypeStartStepRun {
		return w.startStepRun(ctx, assignedAction)
	} else if assignedAction.ActionType == client.ActionTypeCancelStepRun {
		return w.cancelStepRun(ctx, assignedAction)
	}

	return nil, fmt.Errorf("unknown action type: %s", assignedAction.ActionType)
}

func (w *Worker) startStepRun(ctx context.Context, assignedAction *client.Action) (result any, err error) {
	// send a message that the step run started
	_, err = w.client.Dispatcher().SendActionEvent(
		ctx,
		w.getActionEvent(assignedAction, client.ActionEventTypeStarted),
	)

	if err != nil {
		return nil, fmt.Errorf("could not send action event: %w", err)
	}

	action, ok := w.actions[assignedAction.ActionId]

	if !ok {
		return nil, fmt.Errorf("job not found")
	}

	arg, err := decodeArgsToInterface(reflect.TypeOf(action.MethodFn()))

	if err != nil {
		return nil, fmt.Errorf("could not decode args to interface: %w", err)
	}

	err = assignedAction.ActionPayload(arg)

	if err != nil {
		return nil, fmt.Errorf("could not decode action payload: %w", err)
	}

	runContext, cancel := context.WithCancel(context.Background())

	w.cancelMap.Store(assignedAction.StepRunId, cancel)

	runResults := action.Run(runContext, arg)

	// check whether run context was cancelled while action was running
	select {
	case <-runContext.Done():
		w.l.Debug().Msgf("step run %s was cancelled, returning", assignedAction.StepRunId)
		return nil, nil
	default:
	}

	if len(runResults) == 2 {
		result = runResults[0]
	}

	if runResults[len(runResults)-1] != nil {
		err = runResults[1].(error)
	}

	if err != nil {
		failureEvent := w.getActionEvent(assignedAction, client.ActionEventTypeFailed)

		w.alerter.SendAlert(context.Background(), err, map[string]interface{}{
			"actionId":   assignedAction.ActionId,
			"workerId":   assignedAction.WorkerId,
			"stepRunId":  assignedAction.StepRunId,
			"jobName":    assignedAction.JobName,
			"actionType": assignedAction.ActionType,
		})

		failureEvent.EventPayload = err.Error()

		_, err := w.client.Dispatcher().SendActionEvent(
			ctx,
			failureEvent,
		)

		if err != nil {
			return nil, fmt.Errorf("could not send action event: %w", err)
		}

		return nil, err
	}

	// send a message that the step run completed
	finishedEvent, err := w.getActionFinishedEvent(assignedAction, result)

	if err != nil {
		return nil, fmt.Errorf("could not create finished event: %w", err)
	}

	_, err = w.client.Dispatcher().SendActionEvent(
		ctx,
		finishedEvent,
	)

	if err != nil {
		return nil, fmt.Errorf("could not send action event: %w", err)
	}

	return result, nil
}

func (w *Worker) cancelStepRun(ctx context.Context, assignedAction *client.Action) (result any, err error) {
	cancel, ok := w.cancelMap.Load(assignedAction.StepRunId)

	if !ok {
		return nil, fmt.Errorf("could not find step run to cancel")
	}

	w.l.Debug().Msgf("cancelling step run %s", assignedAction.StepRunId)

	cancelFn := cancel.(context.CancelFunc)

	cancelFn()

	return nil, nil
}

func (w *Worker) getActionEvent(action *client.Action, eventType client.ActionEventType) *client.ActionEvent {
	timestamp := time.Now().UTC()

	return &client.ActionEvent{
		Action:         action,
		EventTimestamp: &timestamp,
		EventType:      eventType,
	}
}

func (w *Worker) getActionFinishedEvent(action *client.Action, output any) (*client.ActionEvent, error) {
	event := w.getActionEvent(action, client.ActionEventTypeCompleted)

	outputBytes, err := json.Marshal(output)

	if err != nil {
		return nil, fmt.Errorf("could not marshal step output: %w", err)
	}

	event.EventPayload = string(outputBytes)

	return event, nil
}

func getHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "Unknown"
	}
	return hostName
}
