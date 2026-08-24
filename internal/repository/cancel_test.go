package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// seedBatch stores a submitted batch through the public API so read paths have
// a record to find.
func seedBatch(t *testing.T, store *Store, id string) domain.ReviewBatch {
	t.Helper()
	ctx := context.Background()
	scope, err := domain.NewScope("project-cancel", "material-cancel")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := domain.NewBatch(id, scope, []string{"ev-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := batch.Transition(domain.BatchSubmitted, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	return batch
}

// TestFindBatchCancelledWhileWaitingForLock reproduces the detail-query defect:
// when the caller's context is cancelled while FindBatch is blocked waiting for
// the read lock, the call must surface the cancellation instead of returning the
// full batch as if it were a normal query.
func TestFindBatchCancelledWhileWaitingForLock(t *testing.T) {
	store := NewStore()
	batch := seedBatch(t, store, "batch-cancel")

	// Hold the write lock so the reader must block inside RLock().
	store.mu.Lock()

	// "reader blocked" is signalled once FindBatch is observed to be waiting for
	// the lock; we approximate that by letting the reader run until it has had
	// time to enter RLock(). Since we hold Lock(), RLock() cannot proceed until
	// we release it, guaranteeing the reader is parked when we cancel.
	readerBlocked := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	batchCh := make(chan domain.ReviewBatch, 1)

	go func() {
		close(readerBlocked)
		got, err := store.FindBatch(ctx, batch.ID)
		errCh <- err
		batchCh <- got
	}()

	<-readerBlocked
	// Give the reader a moment to park inside RLock() before we cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()

	// Release the write lock so RLock() can finally acquire. Without the fix,
	// FindBatch would now return the batch as a normal success.
	store.mu.Unlock()

	if err := <-errCh; !errors.Is(err, domain.ErrCancelled) {
		t.Fatalf("expected domain.ErrCancelled after cancelling while waiting for the lock, got %v", err)
	}
	if got := <-batchCh; got.ID != "" {
		t.Fatalf("cancelled detail query must not return batch data, got %+v", got)
	}
}

// TestFindBatchReturnsDataWhenNotCancelled is the control: on a live context
// the same read path returns the stored batch, confirming the fix did not
// change normal behaviour.
func TestFindBatchReturnsDataWhenNotCancelled(t *testing.T) {
	store := NewStore()
	batch := seedBatch(t, store, "batch-live")

	got, err := store.FindBatch(context.Background(), batch.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != batch.ID || got.State != batch.State {
		t.Fatalf("returned batch mismatch: got %+v", got)
	}
}

// TestSaveBatchCancelledWhileWaitingForLock confirms a write path cancelled
// while waiting for the write lock does not mutate the store afterwards.
func TestSaveBatchCancelledWhileWaitingForLock(t *testing.T) {
	store := NewStore()
	original := seedBatch(t, store, "batch-write")

	// Hold the write lock so the writer must block inside Lock().
	store.mu.Lock()

	writerBlocked := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		close(writerBlocked)
		// Same id, higher revision — would overwrite if it were not cancelled.
		next := original
		next.Revision = original.Revision + 1
		next.Reason = "should-not-persist"
		errCh <- store.SaveBatch(ctx, next)
	}()

	<-writerBlocked
	time.Sleep(20 * time.Millisecond)
	cancel()
	store.mu.Unlock()

	if err := <-errCh; !errors.Is(err, domain.ErrCancelled) {
		t.Fatalf("expected domain.ErrCancelled, got %v", err)
	}

	got, err := store.FindBatch(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("unexpected error reading back: %v", err)
	}
	if got.Reason == "should-not-persist" {
		t.Fatalf("cancelled write must not have mutated the store: %+v", got)
	}
}
