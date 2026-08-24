package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

type synchronizedNotificationStore struct {
	*repository.Store
	entered chan struct{}
	release chan struct{}
}

func (s *synchronizedNotificationStore) PendingNotifications(ctx context.Context, limit int) ([]domain.Notification, error) {
	s.entered <- struct{}{}
	<-s.release
	return s.Store.PendingNotifications(ctx, limit)
}

func TestConcurrentDispatchersDeliverOneNotificationOnce(t *testing.T) {
	store := &synchronizedNotificationStore{
		Store:   repository.NewStore(),
		entered: make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	notifier := NewMemoryNotifier()
	first := NewEvidenceService(store, store, store, notifier)
	second := NewEvidenceService(store, store, store, notifier)
	ctx := context.Background()

	evidence, err := first.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-42",
		Kind: domain.Certificate, Supplier: "supplier", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := first.CreateBatch(ctx, CreateBatchCommand{ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.StartReview(ctx, batch.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err := first.DecideBatch(ctx, DecideBatchCommand{BatchID: batch.ID, Approved: true, Actor: "reviewer", Recipient: "ops@example.com"}); err != nil {
		t.Fatal(err)
	}

	var group sync.WaitGroup
	for _, dispatcher := range []*EvidenceService{first, second} {
		group.Add(1)
		go func(service *EvidenceService) {
			defer group.Done()
			if err := service.DispatchPending(ctx, 10); err != nil {
				t.Errorf("dispatch pending: %v", err)
			}
		}(dispatcher)
	}
	for range 2 {
		select {
		case <-store.entered:
		case <-time.After(time.Second):
			t.Fatal("dispatchers did not reach notification lookup")
		}
	}
	close(store.release)
	group.Wait()

	if deliveries := notifier.Deliveries(); len(deliveries) != 1 {
		t.Fatalf("deliveries = %d, want 1", len(deliveries))
	}
}
