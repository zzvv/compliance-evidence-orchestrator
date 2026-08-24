package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

// TestEvidencePreviewMutationDoesNotLeakIntoSavedFacts reproduces the material
// tampering scenario end to end: an operator previews a different expiry on the
// object returned by registration without persisting it, and a second reviewer
// reading the batch detail, the reference lookup, and the report must all see
// the originally saved certificate term.
func TestEvidencePreviewMutationDoesNotLeakIntoSavedFacts(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()

	original := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	saved := original // capture the persisted term in a value the preview mutation cannot reach
	command := RegisterEvidenceCommand{
		ProjectID:  "project-a",
		MaterialID: "material-a",
		Reference:  "CERT-1",
		Kind:       domain.Certificate,
		Supplier:   "supplier",
		IssuedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:  &original,
		Actor:      "operator",
	}
	evidence, err := service.RegisterEvidence(ctx, command)
	if err != nil {
		t.Fatal(err)
	}

	// Preview-only mutation on the object the operator still holds; not saved.
	if evidence.ExpiresAt == nil {
		t.Fatal("registered evidence has no expiry")
	}
	preview := saved.Add(365 * 24 * time.Hour)
	*evidence.ExpiresAt = preview

	scope := domain.Scope{ProjectID: "project-a", MaterialID: "material-a"}

	// Single-item lookup by id (used by batch composition) stays at the saved term.
	found, err := store.FindEvidence(ctx, scope, evidence.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found.ExpiresAt.Equal(saved) {
		t.Fatalf("FindEvidence expiry = %v, want %v", *found.ExpiresAt, saved)
	}

	// Reference lookup (used to locate the same evidence by certificate number) stays saved.
	byRef, err := store.FindEvidenceByReference(ctx, scope, "CERT-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(byRef) != 1 || !byRef[0].ExpiresAt.Equal(saved) {
		t.Fatalf("FindEvidenceByReference expiry = %v, want %v", byRef, saved)
	}

	// List lookup (used by reports and detail views) stays saved.
	listed, err := store.ListEvidence(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || !listed[0].ExpiresAt.Equal(saved) {
		t.Fatalf("ListEvidence expiry = %v, want %v", listed, saved)
	}

	// The compliance report aggregates from the store, so it must report the saved term.
	report, err := service.Report(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Evidence) != 1 || !report.Evidence[0].ExpiresAt.Equal(saved) {
		t.Fatalf("report expiry = %v, want %v", report.Evidence, saved)
	}
}
