package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

var errDecisionReceiptUnavailable = errors.New("decision receipt store is temporarily unavailable")

type decisionReceiptFailureStore struct{ *repository.Store }

func (s decisionReceiptFailureStore) AppendReceipt(ctx context.Context, receipt domain.Receipt) error {
	if receipt.Kind == domain.ReceiptApproved || receipt.Kind == domain.ReceiptRejected {
		return errDecisionReceiptUnavailable
	}
	return s.Store.AppendReceipt(ctx, receipt)
}

func (s decisionReceiptFailureStore) CommitDecision(context.Context, domain.ReviewBatch, domain.Receipt) error {
	return errDecisionReceiptUnavailable
}

func TestFailedDecisionLeavesBatchReviewableAndWithoutDecisionReceipt(t *testing.T) {
	base := repository.NewStore()
	failing := decisionReceiptFailureStore{Store: base}
	service := NewEvidenceService(failing, failing, failing, NewMemoryNotifier())
	ctx := context.Background()

	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-DECISION-100",
		Kind: domain.Certificate, Supplier: "supplier", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{
		ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}, Actor: "operator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartReview(ctx, batch.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}

	_, err = service.DecideBatch(ctx, DecideBatchCommand{BatchID: batch.ID, Approved: true, Actor: "reviewer"})
	if !errors.Is(err, errDecisionReceiptUnavailable) {
		t.Fatalf("DecideBatch error = %v, want decision receipt persistence error", err)
	}

	detail, err := service.BatchDetail(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Batch.State != domain.BatchUnderReview {
		t.Fatalf("failed decision left batch state %s, want %s", detail.Batch.State, domain.BatchUnderReview)
	}
	for _, receipt := range detail.Receipts {
		if receipt.Kind == domain.ReceiptApproved || receipt.Kind == domain.ReceiptRejected {
			t.Fatalf("failed decision left receipt %+v", receipt)
		}
	}
}
