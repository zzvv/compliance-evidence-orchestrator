package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type countingPendingDispatcher struct{ calls atomic.Int32 }

func (d *countingPendingDispatcher) DispatchPending(context.Context, int) error {
	d.calls.Add(1)
	return nil
}

func TestDispatcherDoesNotDispatchAfterShutdownStarts(t *testing.T) {
	service := &countingPendingDispatcher{}
	dispatcher := NewDispatcher(service, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() { dispatcher.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancellation")
	}
	if calls := service.calls.Load(); calls != 0 {
		t.Fatalf("dispatch calls after shutdown = %d, want 0", calls)
	}
}
