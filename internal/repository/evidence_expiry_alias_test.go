package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func TestEvidenceReadsAndInputsDoNotAliasStoredExpiry(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	issuedAt := time.Date(2026, time.January, 12, 0, 0, 0, 0, time.UTC)
	expiresAt := issuedAt.AddDate(1, 0, 0)
	evidence, err := domain.NewEvidence("cert-006", "project-006", "material-006", "ROHS-006", domain.Certificate, "Northstar Metals", issuedAt)
	if err != nil {
		t.Fatalf("new evidence: %v", err)
	}
	evidence.ExpiresAt = &expiresAt
	if err := store.SaveEvidence(ctx, evidence); err != nil {
		t.Fatalf("save evidence: %v", err)
	}

	inputMutation := issuedAt.AddDate(2, 0, 0)
	*evidence.ExpiresAt = inputMutation
	assertPrivateStoredExpiry(t, store, issuedAt.AddDate(1, 0, 0))

	items, err := store.ListEvidence(ctx, evidence.Scope())
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(items) != 1 || items[0].ExpiresAt == nil {
		t.Fatalf("unexpected listed evidence: %#v", items)
	}
	*items[0].ExpiresAt = issuedAt.AddDate(3, 0, 0)
	assertPrivateStoredExpiry(t, store, issuedAt.AddDate(1, 0, 0))

	found, err := store.FindEvidence(ctx, evidence.Scope(), evidence.ID)
	if err != nil {
		t.Fatalf("find evidence: %v", err)
	}
	if found.ExpiresAt == nil {
		t.Fatal("found evidence has no expiry")
	}
	*found.ExpiresAt = issuedAt.AddDate(4, 0, 0)
	assertPrivateStoredExpiry(t, store, issuedAt.AddDate(1, 0, 0))
}

func assertPrivateStoredExpiry(t *testing.T, store *Store, want time.Time) {
	t.Helper()
	stored, err := store.FindEvidence(context.Background(), domain.Scope{ProjectID: "project-006", MaterialID: "material-006"}, "cert-006")
	if err != nil {
		t.Fatalf("find stored evidence: %v", err)
	}
	if stored.ExpiresAt == nil || !stored.ExpiresAt.Equal(want) {
		t.Fatalf("stored expiry = %v, want %v", stored.ExpiresAt, want)
	}
}
