package webhook

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"

	"github.com/rs/zerolog"
)

type WebhookWorker struct {
	opts   WorkerOpts
	client client.Client
	l      *zerolog.Logger
}

type WorkerOpts struct {
	Name      string
	Token     string
	ID        string
	Secret    string
	URL       string
	TenantID  string
	Actions   []string
	WebhookId string
	Logger    *zerolog.Logger
}

func New(opts WorkerOpts) (*WebhookWorker, error) {
	cl, err := client.New(
		client.WithToken(opts.Token),
		client.WithLogger(opts.Logger),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	return &WebhookWorker{
		opts:   opts,
		client: cl,
		l:      opts.Logger,
	}, nil
}

func (w *WebhookWorker) Start() (func() error, error) {
	r, err := worker.NewWorker(
		worker.WithClient(w.client),
		worker.WithInternalData(w.opts.Actions),
		worker.WithName("Webhook_"+w.opts.ID),
		worker.WithLogger(w.l),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create webhook worker: %w", err)
	}

	cleanup, err := r.StartWebhook(worker.WebhookWorkerOpts{
		URL:       w.opts.URL,
		Secret:    w.opts.Secret,
		WebhookId: w.opts.WebhookId,
	})
	if err != nil {
		return nil, fmt.Errorf("could not start webhook worker: %w", err)
	}

	return cleanup, nil
}
