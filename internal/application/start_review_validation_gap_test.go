package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

func TestExpiredEvidenceCannotBeAdvancedPastEligibilityCheck(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	expiresAt := time.Now().Add(120 * time.Millisecond)

	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a",
		MaterialID: "material-a",
		Reference: "CERT-01",
		Kind: domain.Certificate,
		Supplier: "supplier",
		IssuedAt: time.Now().Add(-time.Hour),
		ExpiresAt: &expiresAt,
		Actor: "operator",
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{
		ProjectID: "project-a",
		MaterialID: "material-a",
		EvidenceIDs: []string{evidence.ID},
		Actor: "operator",
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(160 * time.Millisecond)
	allowed, reasons, err := service.CanStartReview(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if allowed || len(reasons) == 0 {
		t.Fatalf("expired evidence must be rejected by eligibility check: allowed=%t reasons=%v", allowed, reasons)
	}
	if _, err := service.StartReview(ctx, batch.ID, "reviewer"); err == nil {
		t.Fatal("expired evidence advanced into review after the eligibility check rejected it")
	}
}
