package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// recordingDispatcher records every DispatchPending invocation and blocks each
// call until the provided release channel is closed, so the test can control
// exactly when the in-flight dispatch returns relative to cancellation.
type recordingDispatcher struct {
	calls   *int32
	release <-chan struct{}
}

func (r *recordingDispatcher) DispatchPending(ctx context.Context, limit int) error {
	atomic.AddInt32(r.calls, 1)
	// Hold the call open until the test signals release; otherwise the loop
	// would race straight back to the select before we can cancel.
	select {
	case <-r.release:
	case <-ctx.Done():
	}
	return ctx.Err()
}

func TestDispatcherDoesNotStartNewDispatchAfterCancel(t *testing.T) {
	var calls int32
	release := make(chan struct{})
	dispatcher := NewDispatcher(&recordingDispatcher{calls: &calls, release: release}, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		dispatcher.Run(ctx)
		close(done)
	}()

	// Wait for the first dispatch cycle to start, then cancel while it is
	// still in flight. Without the shutdown guard the loop would return to
	// the top and start a second dispatch after release.
	if err := waitForCalls(&calls, 1, time.Second); err != nil {
		t.Fatalf("first dispatch did not start: %v", err)
	}
	cancel()
	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancel")
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("dispatch started after cancellation: calls = %d, want 1", got)
	}
}

func TestDispatcherContinuesDispatchDuringNormalOperation(t *testing.T) {
	var calls int32
	release := make(chan struct{})
	close(release) // each dispatch returns immediately
	dispatcher := NewDispatcher(&recordingDispatcher{calls: &calls, release: release}, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		dispatcher.Run(ctx)
		close(done)
	}()

	// During normal operation the ticker should fire several times.
	if err := waitForCalls(&calls, 3, time.Second); err != nil {
		t.Fatalf("normal dispatch did not progress: %v", err)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancel")
	}
}

func waitForCalls(counter *int32, target int32, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		if atomic.LoadInt32(counter) >= target {
			return nil
		}
		select {
		case <-deadline:
			return errTimeout{got: atomic.LoadInt32(counter), want: target}
		case <-time.After(time.Millisecond):
		}
	}
}

type errTimeout struct{ got, want int32 }

func (e errTimeout) Error() string { return "timed out waiting for dispatch calls" }
