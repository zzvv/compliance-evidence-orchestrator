package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

// cancelingNotifier records a delivery then, on the first successful
// delivery, cancels the context it was handed. This reproduces the race where
// the dispatch context is cancelled between a successful delivery and the
// status write-back.
type cancelingNotifier struct {
	mu         sync.Mutex
	deliveries []Delivery
	canceller  context.CancelFunc
}

func (n *cancelingNotifier) Deliver(ctx context.Context, recipient, event string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n.mu.Lock()
	n.deliveries = append(n.deliveries, Delivery{Recipient: recipient, Event: event})
	cancel := n.canceller
	n.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	// The context is now cancelled, but delivery already succeeded.
	return nil
}
func (n *cancelingNotifier) Deliveries() []Delivery {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]Delivery(nil), n.deliveries...)
}

func newPendingNotification(id, batchID, recipient, event string) domain.Notification {
	now := time.Now()
	return domain.Notification{ID: id, BatchID: batchID, Recipient: recipient, Event: event, State: domain.NotificationPending, CreatedAt: now, UpdatedAt: now}
}

// TestDispatchPendingDeliveredStateSurvivesPostDeliveryCancel reproduces the
// cancellation boundary defect: once a notification has been delivered to the
// recipient, a late cancellation must not drop the delivered status, otherwise
// the next dispatch re-sends a duplicate.
func TestDispatchPendingDeliveredStateSurvivesPostDeliveryCancel(t *testing.T) {
	store := repository.NewStore()
	ctx, cancel := context.WithCancel(context.Background())
	notifier := &cancelingNotifier{canceller: cancel}
	service := NewEvidenceService(store, store, store, notifier)

	notification := newPendingNotification("notice-1", "batch-1", "alice", "review_approved")
	if err := store.SaveNotification(context.Background(), notification); err != nil {
		t.Fatal(err)
	}

	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if got := len(notifier.Deliveries()); got != 1 {
		t.Fatalf("expected one delivery, got %d", got)
	}

	// The delivered status must have persisted despite the post-delivery cancel.
	persisted, err := store.PendingNotifications(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 0 {
		t.Fatalf("delivered notification was not persisted as delivered: %v", persisted)
	}

	// A second dispatch must not re-send the notification to the recipient.
	if err := service.DispatchPending(context.Background(), 10); err != nil {
		t.Fatalf("second dispatch: %v", err)
	}
	if got := len(notifier.Deliveries()); got != 1 {
		t.Fatalf("expected delivery count to stay at 1, got %d (duplicate sent)", got)
	}
}

// TestDispatchPendingStopsProcessingOnCancelBeforeStart ensures that, when the
// dispatch context is already cancelled before a notification is picked up,
// processing stops and no delivery is attempted for not-yet-started items.
func TestDispatchPendingStopsProcessingOnCancelBeforeStart(t *testing.T) {
	store := repository.NewStore()
	notifier := NewMemoryNotifier()
	service := NewEvidenceService(store, store, store, notifier)

	for i := 0; i < 2; i++ {
		notification := newPendingNotification("notice-"+string(rune('a'+i)), "batch-1", "alice", "review_approved")
		if err := store.SaveNotification(context.Background(), notification); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.DispatchPending(ctx, 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if got := len(notifier.Deliveries()); got != 0 {
		t.Fatalf("expected no delivery before cancel, got %d", got)
	}

	// The pending notifications are untouched and remain dispatchable later.
	pending, err := store.PendingNotifications(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 still-pending notifications, got %d", len(pending))
	}
}

// TestDispatchPendingFailedNotificationRetries ensures a genuine delivery
// failure (not a cancellation) marks the notification failed and that it is
// retried on the next dispatch once delivery succeeds.
func TestDispatchPendingFailedNotificationRetries(t *testing.T) {
	store := repository.NewStore()
	notifier := NewMemoryNotifier()
	service := NewEvidenceService(store, store, store, notifier)

	notification := newPendingNotification("notice-1", "batch-1", "alice", "review_approved")
	if err := store.SaveNotification(context.Background(), notification); err != nil {
		t.Fatal(err)
	}

	boom := errors.New("transient outage")
	notifier.SetFailure(boom)

	if err := service.DispatchPending(context.Background(), 10); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	// Failed notifications remain pending/retryable.
	pending, err := store.PendingNotifications(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].State != domain.NotificationFailed {
		t.Fatalf("expected 1 failed notification, got %v", pending)
	}
	if got := len(notifier.Deliveries()); got != 0 {
		t.Fatalf("expected no delivery on failure, got %d", got)
	}

	// Recover: the next dispatch delivers and marks the notification delivered.
	notifier.SetFailure(nil)
	if err := service.DispatchPending(context.Background(), 10); err != nil {
		t.Fatalf("retry dispatch: %v", err)
	}
	if got := len(notifier.Deliveries()); got != 1 {
		t.Fatalf("expected delivery after retry, got %d", got)
	}
	pending, err = store.PendingNotifications(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no remaining pending notifications, got %d", len(pending))
	}
}

// TestDispatchPendingNormalDeliveryPersists is a baseline confirming ordinary
// successful delivery is persisted and not re-sent.
func TestDispatchPendingNormalDeliveryPersists(t *testing.T) {
	store := repository.NewStore()
	notifier := NewMemoryNotifier()
	service := NewEvidenceService(store, store, store, notifier)

	notification := newPendingNotification("notice-1", "batch-1", "alice", "review_approved")
	if err := store.SaveNotification(context.Background(), notification); err != nil {
		t.Fatal(err)
	}

	if err := service.DispatchPending(context.Background(), 10); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if got := len(notifier.Deliveries()); got != 1 {
		t.Fatalf("expected one delivery, got %d", got)
	}
	if err := service.DispatchPending(context.Background(), 10); err != nil {
		t.Fatalf("second dispatch: %v", err)
	}
	if got := len(notifier.Deliveries()); got != 1 {
		t.Fatalf("expected delivery count to stay at 1, got %d", got)
	}
}
