package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

func TestWorkPlanOmitsClosedBatchesUnlessRequested(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	now := time.Now().UTC()
	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-1", Kind: domain.Certificate, Supplier: "supplier", IssuedAt: now.Add(-time.Hour)})
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
	scope := domain.Scope{ProjectID: "project-a", MaterialID: "material-a"}
	plan, err := service.WorkPlan(ctx, scope, WorkPlanOptions{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 0 {
		t.Fatalf("closed batch actions = %+v", plan.Actions)
	}
	plan, err = service.WorkPlan(ctx, scope, WorkPlanOptions{Now: now, IncludeClosed: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Batches.Completed != 1 {
		t.Fatalf("completed = %d", plan.Batches.Completed)
	}
}

func TestParseWorkPlanOptions(t *testing.T) {
	options, err := ParseWorkPlanOptions("include_closed")
	if err != nil || !options.IncludeClosed {
		t.Fatalf("options = %+v, err = %v", options, err)
	}
	if _, err := ParseWorkPlanOptions("unknown"); err == nil {
		t.Fatal("expected invalid option error")
	}
}
