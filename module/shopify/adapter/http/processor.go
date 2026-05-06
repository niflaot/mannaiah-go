package http

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyorderservice "mannaiah/module/shopify/application/order/service"

	"go.uber.org/zap"
)

var (
	// ErrNilProcessorContactsService is returned when a nil contact sync service is provided.
	ErrNilProcessorContactsService = errors.New("shopify webhook contact sync service must not be nil")
	// ErrNilProcessorOrdersService is returned when a nil order sync service is provided.
	ErrNilProcessorOrdersService = errors.New("shopify webhook order sync service must not be nil")
	// ErrProcessorClosed is returned when webhook jobs are enqueued after shutdown.
	ErrProcessorClosed = errors.New("shopify webhook processor is closed")
)

// WebhookProcessor defines asynchronous Shopify webhook processing behavior.
type WebhookProcessor interface {
	// Enqueue schedules one Shopify webhook job.
	Enqueue(ctx context.Context, topic string, shopifyID string) error
}

type webhookJob struct {
	topic     string
	shopifyID string
}

// Processor defines asynchronous Shopify webhook processing behavior.
type Processor struct {
	// contactsService defines contact sync dependencies.
	contactsService shopifycontactservice.Service
	// ordersService defines order sync dependencies.
	ordersService shopifyorderservice.Service
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// timeout defines per-job execution timeout values.
	timeout time.Duration
	// jobs defines queued webhook work.
	jobs chan webhookJob
	// once defines shutdown synchronization.
	once sync.Once
	// closed defines close-state synchronization.
	closed chan struct{}
	// workers defines worker wait groups.
	workers sync.WaitGroup
}

var (
	// _ ensures Processor satisfies webhook processor contracts.
	_ WebhookProcessor = (*Processor)(nil)
)

// NewProcessor creates Shopify webhook processors and starts background workers immediately.
func NewProcessor(workers int, timeout time.Duration, contactsService shopifycontactservice.Service, ordersService shopifyorderservice.Service, providedLogger *zap.Logger) (*Processor, error) {
	if contactsService == nil {
		return nil, ErrNilProcessorContactsService
	}
	if ordersService == nil {
		return nil, ErrNilProcessorOrdersService
	}
	if workers <= 0 {
		workers = 1
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	processor := &Processor{
		contactsService: contactsService,
		ordersService:   ordersService,
		logger:          logger,
		timeout:         timeout,
		jobs:            make(chan webhookJob, workers*8),
		closed:          make(chan struct{}),
	}
	for index := 0; index < workers; index++ {
		processor.workers.Add(1)
		go processor.runWorker()
	}

	return processor, nil
}

// Enqueue schedules one Shopify webhook job.
func (p *Processor) Enqueue(ctx context.Context, topic string, shopifyID string) error {
	job := webhookJob{topic: strings.TrimSpace(topic), shopifyID: strings.TrimSpace(shopifyID)}
	select {
	case <-p.closed:
		return ErrProcessorClosed
	case <-ctx.Done():
		return ctx.Err()
	case p.jobs <- job:
		return nil
	}
}

// Stop shuts down webhook workers gracefully.
func (p *Processor) Stop(ctx context.Context) error {
	p.once.Do(func() {
		close(p.closed)
		close(p.jobs)
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.workers.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (p *Processor) runWorker() {
	defer p.workers.Done()
	for job := range p.jobs {
		if strings.TrimSpace(job.shopifyID) == "" {
			continue
		}
		jobCtx, cancel := context.WithTimeout(context.Background(), p.timeout)
		p.processJob(jobCtx, job)
		cancel()
	}
}

func (p *Processor) processJob(ctx context.Context, job webhookJob) {
	topic := strings.ToLower(strings.TrimSpace(job.topic))
	var err error
	if isCustomerTopic(topic) {
		_, err = p.contactsService.SyncContactByID(ctx, "webhook", job.shopifyID)
	} else if isOrderTopic(topic) {
		_, err = p.ordersService.SyncOrderByID(ctx, "webhook", job.shopifyID)
	}
	if err != nil {
		p.logger.Warn("process shopify webhook failed", zap.String("topic", job.topic), zap.String("shopify_id", job.shopifyID), zap.Error(err))
	}
}

func isCustomerTopic(topic string) bool {
	switch topic {
	case "customers/create", "customers/update", "customers/enable", "customers/disable":
		return true
	default:
		return false
	}
}

func isOrderTopic(topic string) bool {
	switch topic {
	case "orders/create", "orders/updated", "orders/paid", "orders/cancelled", "orders/fulfilled":
		return true
	default:
		return false
	}
}
