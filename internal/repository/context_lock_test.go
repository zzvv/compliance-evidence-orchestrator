package repository

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func TestFindBatchRespectsCancellationWhileWaitingForReadLock(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	batch, err := domain.NewBatch("batch-1", domain.Scope{ProjectID: "project-a", MaterialID: "material-a"}, []string{"evidence-1"}, time.Now())
	if err != nil { t.Fatal(err) }
	if err := store.SaveBatch(ctx, batch); err != nil { t.Fatal(err) }

	store.mu.Lock()
	canceled, cancel := context.WithCancel(ctx)
	result := make(chan error, 1)
	go func() { _, err := store.FindBatch(canceled, batch.ID); result <- err }()
	for index := 0; index < 100; index++ { runtime.Gosched() }
	cancel()
	store.mu.Unlock()
	select {
	case err := <-result:
		if !errors.Is(err, domain.ErrCancelled) { t.Fatalf("FindBatch error = %v, want cancellation", err) }
	case <-time.After(time.Second): t.Fatal("FindBatch did not return")
	}
}
