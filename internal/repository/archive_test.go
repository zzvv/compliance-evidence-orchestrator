package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// helper that wires up a terminal batch with a pending notification and an audit
// event scoped to the same project/material, plus a sibling batch that must
// survive any archival sweep.
type archiveFixture struct {
	store   *Store
	scope   domain.Scope
	batch   domain.ReviewBatch
	sibling domain.ReviewBatch
	notice  domain.Notification
	event   domain.AuditEvent
}

func newArchiveFixture(t *testing.T) archiveFixture {
	t.Helper()
	store := NewStore()
	ctx := context.Background()
	scope := domain.Scope{ProjectID: "project-a", MaterialID: "material-a"}

	now := time.Now().UTC()
	batch, err := domain.NewBatch("batch-1", scope, []string{"ev-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if batch, err = transitionTo(batch, domain.BatchApproved, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	sibling, err := domain.NewBatch("batch-2", scope, []string{"ev-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if sibling, err = transitionTo(sibling, domain.BatchSubmitted, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBatch(ctx, sibling); err != nil {
		t.Fatal(err)
	}

	notice := domain.Notification{ID: "notice-1", BatchID: batch.ID, Recipient: "ops@example.com", Event: "review_approved", State: domain.NotificationPending, CreatedAt: now, UpdatedAt: now}
	if err := store.SaveNotification(ctx, notice); err != nil {
		t.Fatal(err)
	}
	siblingNotice := domain.Notification{ID: "notice-2", BatchID: sibling.ID, Recipient: "ops@example.com", Event: "review_submitted", State: domain.NotificationPending, CreatedAt: now, UpdatedAt: now}
	if err := store.SaveNotification(ctx, siblingNotice); err != nil {
		t.Fatal(err)
	}

	event := domain.NewAuditEvent("audit-1", scope, batch.ID, "review_approved", "reviewer", now)
	if err := store.AppendAudit(ctx, event); err != nil {
		t.Fatal(err)
	}
	siblingEvent := domain.NewAuditEvent("audit-2", scope, sibling.ID, "batch_submitted", "supplier", now)
	if err := store.AppendAudit(ctx, siblingEvent); err != nil {
		t.Fatal(err)
	}

	return archiveFixture{store: store, scope: scope, batch: batch, sibling: sibling, notice: notice, event: event}
}

// transitionTo walks a draft batch through its legal workflow path so the test
// can land directly on a terminal or in-flight state.
func transitionTo(batch domain.ReviewBatch, target domain.BatchState, now time.Time) (domain.ReviewBatch, error) {
	steps := map[domain.BatchState][]domain.BatchState{
		domain.BatchSubmitted:   {domain.BatchSubmitted},
		domain.BatchUnderReview: {domain.BatchSubmitted, domain.BatchUnderReview},
		domain.BatchApproved:    {domain.BatchSubmitted, domain.BatchUnderReview, domain.BatchApproved},
		domain.BatchRejected:    {domain.BatchSubmitted, domain.BatchUnderReview, domain.BatchRejected},
		domain.BatchCancelled:   {domain.BatchCancelled},
	}[target]
	for _, step := range steps {
		if err := batch.Transition(step, "", now); err != nil {
			return batch, err
		}
	}
	return batch, nil
}

func TestArchiveBatchPurgesPendingNotificationsAndAudit(t *testing.T) {
	fixture := newArchiveFixture(t)
	ctx := context.Background()

	if err := fixture.store.DeleteBatch(ctx, fixture.batch.ID); err != nil {
		t.Fatalf("delete batch: %v", err)
	}

	// archived batch must no longer be retrievable
	if _, err := fixture.store.FindBatch(ctx, fixture.batch.ID); err == nil {
		t.Fatal("archived batch should be gone")
	}

	// pending notifications for the archived batch must be purged
	pending, err := fixture.store.PendingNotifications(ctx, 100)
	if err != nil {
		t.Fatalf("pending notifications: %v", err)
	}
	for _, n := range pending {
		if n.BatchID == fixture.batch.ID {
			t.Fatalf("archived batch still has pending notification %s", n.ID)
		}
	}

	// audit events referencing the archived batch must be removed from the scope trail
	audit, err := fixture.store.ListAudit(ctx, fixture.scope)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	for _, event := range audit {
		if event.BatchID == fixture.batch.ID {
			t.Fatalf("archived batch still has audit event %s", event.ID)
		}
	}
}

func TestArchiveBatchLeavesSiblingBatchIntact(t *testing.T) {
	fixture := newArchiveFixture(t)
	ctx := context.Background()

	if err := fixture.store.DeleteBatch(ctx, fixture.batch.ID); err != nil {
		t.Fatalf("delete batch: %v", err)
	}

	// sibling batch and its receipts must remain queryable
	if _, err := fixture.store.FindBatch(ctx, fixture.sibling.ID); err != nil {
		t.Fatalf("sibling batch must remain after archiving its peer: %v", err)
	}
	batches, err := fixture.store.ListBatches(ctx, fixture.scope)
	if err != nil {
		t.Fatalf("list batches: %v", err)
	}
	if len(batches) != 1 || batches[0].ID != fixture.sibling.ID {
		t.Fatalf("expected only the sibling batch to remain, got %+v", batches)
	}

	// sibling pending notifications must survive
	pending, err := fixture.store.PendingNotifications(ctx, 100)
	if err != nil {
		t.Fatalf("pending notifications: %v", err)
	}
	var survived bool
	for _, n := range pending {
		if n.BatchID == fixture.sibling.ID {
			survived = true
		}
	}
	if !survived {
		t.Fatal("sibling batch pending notification must survive archiving")
	}

	// sibling audit events must survive
	audit, err := fixture.store.ListAudit(ctx, fixture.scope)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	var auditSurvived bool
	for _, event := range audit {
		if event.BatchID == fixture.sibling.ID {
			auditSurvived = true
		}
	}
	if !auditSurvived {
		t.Fatal("sibling batch audit event must survive archiving")
	}
}

func TestArchiveBatchRejectsUnknownID(t *testing.T) {
	store := NewStore()
	if err := store.DeleteBatch(context.Background(), "missing"); err == nil {
		t.Fatal("deleting unknown batch should error")
	}
}

// SweepExpiredBatches is driven through the application layer; this test
// confirms the retention rule picks terminal batches past their keep window.
func TestRetentionRuleFlagsTerminalBatchesOnly(t *testing.T) {
	rule := domain.DefaultRetentionRule()
	now := time.Now().UTC()

	approved, err := domain.NewBatch("b", domain.Scope{ProjectID: "p", MaterialID: "m"}, []string{"e"}, now.AddDate(-2, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if approved, err = transitionTo(approved, domain.BatchApproved, now.AddDate(-2, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if !rule.IsExpired(approved, now) {
		t.Fatal("two-year-old approved batch should be expired")
	}

	fresh, err := domain.NewBatch("b", domain.Scope{ProjectID: "p", MaterialID: "m"}, []string{"e"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if fresh, err = transitionTo(fresh, domain.BatchApproved, now); err != nil {
		t.Fatal(err)
	}
	if rule.IsExpired(fresh, now) {
		t.Fatal("fresh approved batch should not be expired")
	}
}
