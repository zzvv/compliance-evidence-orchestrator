package worker

import (
	"context"
	"log"
	"time"
)

type PendingDispatcher interface {
	DispatchPending(context.Context, int) error
}
type Dispatcher struct {
	service  PendingDispatcher
	interval time.Duration
}

func NewDispatcher(service PendingDispatcher, interval time.Duration) *Dispatcher {
	if interval <= 0 {
		interval = time.Second
	}
	return &Dispatcher{service: service, interval: interval}
}
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		// If cancellation has already begun, do not start a new dispatch
		// cycle: a delivery implementation that does not check the context
		// in time would otherwise make an external call during shutdown.
		if ctx.Err() != nil {
			return
		}
		if err := d.service.DispatchPending(ctx, 20); err != nil && ctx.Err() == nil {
			log.Printf("notification dispatch: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
