package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

var errReceiptPersistence = errors.New("receipt store is temporarily unavailable")

type submissionFailureRepository struct{ *repository.Store }

func (r submissionFailureRepository) AppendReceipt(context.Context, domain.Receipt) error {
	return errReceiptPersistence
}
func (r submissionFailureRepository) CommitSubmission(context.Context, domain.ReviewBatch, domain.Receipt) error {
	return errReceiptPersistence
}

func TestFailedSubmissionLeavesNoVisibleBatch(t *testing.T) {
	base := repository.NewStore()
	failing := submissionFailureRepository{Store: base}
	service := NewEvidenceService(base, failing, failing, NewMemoryNotifier())
	ctx := context.Background()
	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-100", Kind: domain.Certificate, Supplier: "supplier", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateBatch(ctx, CreateBatchCommand{ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}, Actor: "operator"})
	if !errors.Is(err, errReceiptPersistence) {
		t.Fatalf("CreateBatch error = %v, want receipt persistence error", err)
	}
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	batches, err := base.ListBatches(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 0 {
		t.Fatalf("failed submission left %d visible batch(es): %#v", len(batches), batches)
	}
}
