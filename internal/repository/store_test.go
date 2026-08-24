package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"testing"
	"time"
)

func TestEvidenceIsScopedByProjectAndMaterial(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	first, err := domain.NewEvidence("ev-a", "project-a", "material-a", "CERT-1", domain.Certificate, "one", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.NewEvidence("ev-a", "project-b", "material-b", "CERT-1", domain.Certificate, "two", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEvidence(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEvidence(ctx, second); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindEvidence(ctx, second.Scope(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Supplier != "two" {
		t.Fatalf("supplier = %s", got.Supplier)
	}
}
