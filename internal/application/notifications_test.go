package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

// countingNotifier records every delivery so a duplicate delivery across
// concurrent dispatcher instances is observable. A short delay widens the
// window in which two instances read the same pending snapshot.
type countingNotifier struct {
	mu         sync.Mutex
	deliveries []string
}

func (n *countingNotifier) Deliver(ctx context.Context, recipient, event string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	time.Sleep(2 * time.Millisecond)
	n.mu.Lock()
	defer n.mu.Unlock()
	n.deliveries = append(n.deliveries, recipient)
	return nil
}
func (n *countingNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.deliveries)
}

// TestDispatchPendingDoesNotDoubleDeliver simulates two scheduler instances
// sharing the same store and processing the same pending notifications at
// the same time. Each notification must be delivered exactly once.
func TestDispatchPendingDoesNotDoubleDeliver(t *testing.T) {
	for round := 0; round < 20; round++ {
		store := repository.NewStore()
		notifier := &countingNotifier{}
		ctx := context.Background()

		// Two dispatch instances share one store and one notifier.
		svcA := NewEvidenceService(store, store, store, notifier)
		svcB := NewEvidenceService(store, store, store, notifier)

		now := time.Now()
		const total = 5
		for i := 0; i < total; i++ {
			n := domain.Notification{
				ID:        fmt.Sprintf("notice-%d", i),
				BatchID:   "batch-x",
				Recipient: "reviewer@example.com",
				Event:     "review_approved",
				State:     domain.NotificationPending,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := store.SaveNotification(ctx, n); err != nil {
				t.Fatal(err)
			}
		}

		// Barrier so both instances enter DispatchPending at the same instant.
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		errs := make([]error, 2)
		run := func(idx int, svc *EvidenceService) {
			defer wg.Done()
			<-start
			errs[idx] = svc.DispatchPending(ctx, 20)
		}
		go run(0, svcA)
		go run(1, svcB)
		close(start)
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d instance %d: %v", round, i, err)
			}
		}

		if got := notifier.count(); got != total {
			t.Fatalf("round %d: deliveries = %d, want %d (duplicate delivery)", round, got, total)
		}

		remaining, err := store.PendingNotifications(ctx, 20)
		if err != nil {
			t.Fatal(err)
		}
		if len(remaining) != 0 {
			t.Fatalf("round %d: pending notifications left = %d, want 0", round, len(remaining))
		}
	}
}

// TestDispatchPendingRetriesFailedNotifications ensures that a previously
// failed notification is still picked up and retried after the claim gate.
func TestDispatchPendingRetriesFailedNotifications(t *testing.T) {
	store := repository.NewStore()
	notifier := &countingNotifier{}
	ctx := context.Background()
	svc := NewEvidenceService(store, store, store, notifier)

	now := time.Now()
	n := domain.Notification{
		ID:        "notice-retry",
		BatchID:   "batch-x",
		Recipient: "reviewer@example.com",
		Event:     "review_approved",
		State:     domain.NotificationFailed,
		Attempts:  1,
		LastError: "transient",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.SaveNotification(ctx, n); err != nil {
		t.Fatal(err)
	}

	if err := svc.DispatchPending(ctx, 20); err != nil {
		t.Fatal(err)
	}

	if got := notifier.count(); got != 1 {
		t.Fatalf("deliveries = %d, want 1", got)
	}
	remaining, err := store.PendingNotifications(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("pending notifications left = %d, want 0", len(remaining))
	}
}
