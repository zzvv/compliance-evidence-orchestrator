package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

func TestDeletingBatchRemovesPendingNotificationsAndItsAudit(t *testing.T) {
	ctx := context.Background()
	store := repository.NewStore()
	notifier := NewMemoryNotifier()
	service := NewEvidenceService(store, store, store, notifier)
	scope := domain.Scope{ProjectID: "project-007", MaterialID: "material-007"}
	now := time.Date(2026, time.February, 7, 9, 0, 0, 0, time.UTC)
	batch, err := domain.NewBatch("batch-expired", scope, []string{"evidence-007"}, now)
	if err != nil {
		t.Fatalf("new batch: %v", err)
	}
	if err := batch.Transition(domain.BatchSubmitted, "", now); err != nil {
		t.Fatalf("submit batch: %v", err)
	}
	if err := store.SaveBatch(ctx, batch); err != nil {
		t.Fatalf("save batch: %v", err)
	}
	if err := store.SaveNotification(ctx, domain.Notification{ID: "notice-expired", BatchID: batch.ID, Recipient: "review@example.test", Event: "review_submitted", State: domain.NotificationPending, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("save pending notification: %v", err)
	}
	if err := store.AppendAudit(ctx, domain.NewAuditEvent("audit-expired", scope, batch.ID, "batch_submitted", "operator", now)); err != nil {
		t.Fatalf("append batch audit: %v", err)
	}
	if err := store.AppendAudit(ctx, domain.NewAuditEvent("audit-live", scope, "batch-live", "batch_submitted", "operator", now)); err != nil {
		t.Fatalf("append unrelated audit: %v", err)
	}

	if err := store.DeleteBatch(ctx, batch.ID); err != nil {
		t.Fatalf("delete expired batch: %v", err)
	}
	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatalf("dispatch pending: %v", err)
	}
	if deliveries := notifier.Deliveries(); len(deliveries) != 0 {
		t.Fatalf("deleted batch still produced deliveries: %#v", deliveries)
	}

	audit, err := store.ListAudit(ctx, scope)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(audit) != 1 || audit[0].BatchID != "batch-live" {
		t.Fatalf("audit after delete = %#v, want only unrelated batch audit", audit)
	}
}
