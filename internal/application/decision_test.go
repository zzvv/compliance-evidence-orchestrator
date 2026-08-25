package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

// newBatchUnderReview wires a batch through submit + start review, returning
// the batch id ready for a decision.
func newBatchUnderReview(t *testing.T, ctx context.Context, service *EvidenceService) string {
	t.Helper()
	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a", MaterialID: "material-a",
		Reference: "CERT-01", Kind: domain.Certificate,
		Supplier: "supplier", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{
		ProjectID: "project-a", MaterialID: "material-a",
		EvidenceIDs: []string{evidence.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.StartReview(ctx, batch.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	return batch.ID
}

// TestDecideBatchAtomicRollsBackOnReceiptFailure exercises the atomic
// DecisionCommitter path for an approval: when the receipt write fails, the
// batch must not be left in a terminal state without its decision receipt, and
// a retry must complete the decision normally.
func TestDecideBatchAtomicRollsBackOnReceiptFailure(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	batchID := newBatchUnderReview(t, ctx, service)

	// Cancel the context so CommitDecision aborts before touching store state,
	// simulating a receipt write that fails mid-operation.
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.DecideBatch(cancelCtx, DecideBatchCommand{
		BatchID: batchID, Approved: true, Actor: "reviewer",
	}); err == nil {
		t.Fatal("expected DecideBatch to fail when the receipt write fails")
	}

	detail, err := service.BatchDetail(ctx, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Batch.State != domain.BatchUnderReview {
		t.Fatalf("state = %s, want under_review (decision must not persist)", detail.Batch.State)
	}
	for _, receipt := range detail.Receipts {
		if receipt.Kind == domain.ReceiptApproved {
			t.Fatalf("decision receipt should not have been recorded, got %v", receipt)
		}
	}

	if _, err := service.DecideBatch(ctx, DecideBatchCommand{
		BatchID: batchID, Approved: true, Actor: "reviewer",
	}); err != nil {
		t.Fatalf("retry DecideBatch: %v", err)
	}
	detail, err = service.BatchDetail(ctx, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Batch.State != domain.BatchApproved {
		t.Fatalf("state = %s, want approved", detail.Batch.State)
	}
	if len(detail.Receipts) != 3 {
		t.Fatalf("receipts = %d, want 3 after successful retry", len(detail.Receipts))
	}
	if detail.Receipts[len(detail.Receipts)-1].Kind != domain.ReceiptApproved {
		t.Fatalf("last receipt = %s, want approved", detail.Receipts[len(detail.Receipts)-1].Kind)
	}
}

// TestDecideBatchAtomicRollsBackOnRejection mirrors the approval case for a
// rejection, ensuring both decision kinds roll back atomically on failure.
func TestDecideBatchAtomicRollsBackOnRejection(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	batchID := newBatchUnderReview(t, ctx, service)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.DecideBatch(cancelCtx, DecideBatchCommand{
		BatchID: batchID, Approved: false, Reason: "nonconforming", Actor: "reviewer",
	}); err == nil {
		t.Fatal("expected DecideBatch to fail when the receipt write fails")
	}

	detail, err := service.BatchDetail(ctx, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Batch.State != domain.BatchUnderReview {
		t.Fatalf("state = %s, want under_review", detail.Batch.State)
	}

	if _, err := service.DecideBatch(ctx, DecideBatchCommand{
		BatchID: batchID, Approved: false, Reason: "nonconforming", Actor: "reviewer",
	}); err != nil {
		t.Fatalf("retry DecideBatch: %v", err)
	}
	detail, err = service.BatchDetail(ctx, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Batch.State != domain.BatchRejected {
		t.Fatalf("state = %s, want rejected", detail.Batch.State)
	}
	if detail.Receipts[len(detail.Receipts)-1].Kind != domain.ReceiptRejected {
		t.Fatalf("last receipt = %s, want rejected", detail.Receipts[len(detail.Receipts)-1].Kind)
	}
}
