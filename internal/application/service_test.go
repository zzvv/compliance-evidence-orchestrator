package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

func TestReviewLifecycleRecordsReceipts(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-01", Kind: domain.Certificate, Supplier: "supplier", IssuedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.StartReview(ctx, batch.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.DecideBatch(ctx, DecideBatchCommand{BatchID: batch.ID, Approved: true, Actor: "reviewer"}); err != nil {
		t.Fatal(err)
	}
	detail, err := service.BatchDetail(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Batch.State != domain.BatchApproved {
		t.Fatalf("state = %s", detail.Batch.State)
	}
	if len(detail.Receipts) != 3 {
		t.Fatalf("receipts = %d", len(detail.Receipts))
	}
}
