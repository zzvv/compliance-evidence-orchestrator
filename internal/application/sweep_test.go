package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

// TestArchiveSweepClearsDispatchAndAudit exercises the full regression path
// described in BUG_REPRO: once a batch passes its retention window and is swept,
// the dispatcher must not re-issue its queued notification and the per-scope
// audit trail must no longer mix in the dead batch — while a sibling batch's
// notification, audit and review flow remain untouched.
func TestArchiveSweepClearsDispatchAndAudit(t *testing.T) {
	store := repository.NewStore().WithRetentionRule(domain.RetentionRule{
		KeepApprovedFor:  10 * time.Millisecond,
		KeepRejectedFor:  10 * time.Millisecond,
		KeepCancelledFor: 10 * time.Millisecond,
	})
	notifier := NewMemoryNotifier()
	service := NewEvidenceService(store, store, store, notifier)
	ctx := context.Background()

	// register evidence and a sibling batch that must survive the sweep.
	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-01", Kind: domain.Certificate, Supplier: "supplier", IssuedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}, Actor: "supplier"})
	if err != nil {
		t.Fatal(err)
	}
	sibling, err := service.CreateBatch(ctx, CreateBatchCommand{ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}, Actor: "supplier"})
	if err != nil {
		t.Fatal(err)
	}

	// drive the primary batch to approval with a queued notification; leave the
	// sibling mid-review so it still has work pending after the sweep.
	if _, err := service.StartReview(ctx, batch.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.DecideBatch(ctx, DecideBatchCommand{BatchID: batch.ID, Approved: true, Actor: "reviewer", Recipient: "ops@example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartReview(ctx, sibling.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}

	// the primary batch carries one pending notification; the sibling has none.
	pending, err := store.PendingNotifications(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].BatchID != batch.ID {
		t.Fatalf("expected one pending notification for the primary batch, got %+v", pending)
	}

	// wait for the retention window to elapse, then sweep expired batches.
	time.Sleep(20 * time.Millisecond)
	result, err := service.SweepExpiredBatches(ctx, time.Now())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(result.ArchivedBatchIDs) != 1 || result.ArchivedBatchIDs[0] != batch.ID {
		t.Fatalf("expected only the primary batch archived, got %+v", result.ArchivedBatchIDs)
	}

	// the archived batch must no longer be addressable.
	if _, err := service.BatchDetail(ctx, batch.ID); err == nil {
		t.Fatal("archived batch should no longer be retrievable")
	}

	// dispatching after the sweep must not deliver the archived batch's
	// notification, and nothing should be left pending for it.
	if err := service.DispatchPending(ctx, 100); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	deliveries := notifier.Deliveries()
	for _, delivery := range deliveries {
		if delivery.Event == "review_approved" {
			t.Fatalf("archived batch notification was delivered: %+v", delivery)
		}
	}
	pending, err = store.PendingNotifications(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range pending {
		if n.BatchID == batch.ID {
			t.Fatalf("archived batch still has pending notification %s", n.ID)
		}
	}

	// the per-scope audit trail must no longer reference the archived batch, and
	// must still hold the sibling's history.
	scope := domain.Scope{ProjectID: "project-a", MaterialID: "material-a"}
	audit, err := service.AuditTrail(ctx, scope)
	if err != nil {
		t.Fatalf("audit trail: %v", err)
	}
	for _, event := range audit {
		if event.BatchID == batch.ID {
			t.Fatalf("archived batch still appears in audit trail: %+v", event)
		}
	}
	var siblingAuditSeen bool
	for _, event := range audit {
		if event.BatchID == sibling.ID {
			siblingAuditSeen = true
		}
	}
	if !siblingAuditSeen {
		t.Fatal("sibling batch audit trail must survive the sweep")
	}

	// the sibling batch must remain fully operational: still queryable and able
	// to complete its review without touching the archived peer.
	detail, err := service.BatchDetail(ctx, sibling.ID)
	if err != nil {
		t.Fatalf("sibling batch detail: %v", err)
	}
	if detail.Batch.State != domain.BatchUnderReview {
		t.Fatalf("sibling state = %s", detail.Batch.State)
	}
	if _, err := service.DecideBatch(ctx, DecideBatchCommand{BatchID: sibling.ID, Approved: false, Actor: "reviewer", Recipient: "ops@example.com"}); err != nil {
		t.Fatalf("sibling decide: %v", err)
	}
}
