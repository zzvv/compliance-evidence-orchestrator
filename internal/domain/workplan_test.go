package domain

import (
	"testing"
	"time"
)

func TestBuildWorkPlanOrdersUrgentEvidenceBeforeReviews(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	scope := Scope{ProjectID: "project-a", MaterialID: "material-a"}
	expires := now.Add(-time.Hour)
	evidence := Evidence{ID: "evidence-1", ProjectID: scope.ProjectID, MaterialID: scope.MaterialID, Reference: "CERT-1", Kind: Certificate, Supplier: "supplier", IssuedAt: now.Add(-24 * time.Hour), ExpiresAt: &expires}
	batch := ReviewBatch{ID: "batch-1", Scope: scope, EvidenceIDs: []string{evidence.ID}, State: BatchSubmitted, Revision: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	plan, err := BuildWorkPlan(scope, []Evidence{evidence}, []ReviewBatch{batch}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %d", len(plan.Actions))
	}
	if plan.Actions[0].Kind != WorkActionReplaceExpiredEvidence {
		t.Fatalf("first action = %s", plan.Actions[0].Kind)
	}
	if plan.Evidence.Expired != 1 || plan.Batches.Submitted != 1 {
		t.Fatalf("summary = %+v / %+v", plan.Evidence, plan.Batches)
	}
}

func TestBuildWorkPlanRejectsForeignScopeData(t *testing.T) {
	scope := Scope{ProjectID: "project-a", MaterialID: "material-a"}
	foreign := Evidence{ID: "evidence-1", ProjectID: "project-b", MaterialID: "material-a", Reference: "CERT-1", Kind: Certificate, Supplier: "supplier", IssuedAt: time.Now()}
	if _, err := BuildWorkPlan(scope, []Evidence{foreign}, nil, time.Now()); err == nil {
		t.Fatal("expected scope validation error")
	}
}
