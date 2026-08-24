package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// TestEvidenceExpiryIsIsolatedFromCallerMutation guards the data-tampering
// regression: once evidence is saved, no mutation of the object handed back to
// a caller, of a list/detail query result, or of a single-item query result may
// alter the certificate expiry persisted in the store.
func TestEvidenceExpiryIsIsolatedFromCallerMutation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	original := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	registered, err := store.SaveEvidenceReturning(ctx, "project-a", "material-a", "ev-1", "CERT-1", original)
	if err != nil {
		t.Fatal(err)
	}

	// Vector 1: mutate the ExpiresAt pointer on the object the caller still holds.
	tampered := registered.ExpiresAt.Add(365 * 24 * time.Hour)
	*registered.ExpiresAt = tampered
	assertStoredExpiry(t, store, "ev-1", original, "after mutating registered object")

	// Vector 2: mutate an item returned by the list query.
	items, err := store.ListEvidence(ctx, registered.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("list = %d items", len(items))
	}
	if items[0].ExpiresAt == nil {
		t.Fatal("list result has no expiry")
	}
	*items[0].ExpiresAt = tampered
	assertStoredExpiry(t, store, "ev-1", original, "after mutating list result")

	// Vector 3: mutate the single-item query result.
	found, err := store.FindEvidence(ctx, registered.Scope(), "ev-1")
	if err != nil {
		t.Fatal(err)
	}
	if found.ExpiresAt == nil {
		t.Fatal("find result has no expiry")
	}
	*found.ExpiresAt = tampered
	assertStoredExpiry(t, store, "ev-1", original, "after mutating find result")

	// Vector 4: mutate a result returned by reference lookup.
	byRef, err := store.FindEvidenceByReference(ctx, registered.Scope(), "CERT-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(byRef) != 1 {
		t.Fatalf("reference lookup = %d items", len(byRef))
	}
	if byRef[0].ExpiresAt == nil {
		t.Fatal("reference result has no expiry")
	}
	*byRef[0].ExpiresAt = tampered
	assertStoredExpiry(t, store, "ev-1", original, "after mutating reference result")
}

func assertStoredExpiry(t *testing.T, store *Store, id string, want time.Time, stage string) {
	t.Helper()
	got, err := store.FindEvidence(context.Background(), domain.Scope{ProjectID: "project-a", MaterialID: "material-a"}, id)
	if err != nil {
		t.Fatalf("%s: find: %v", stage, err)
	}
	if got.ExpiresAt == nil {
		t.Fatalf("%s: stored expiry is nil", stage)
	}
	if !got.ExpiresAt.Equal(want) {
		t.Fatalf("%s: stored expiry = %v, want %v", stage, *got.ExpiresAt, want)
	}
}

// SaveEvidenceReturning registers a certificate with an expiry and returns the
// value handed back to the caller, exercising the same aliasing path the
// application service exposes.
func (s *Store) SaveEvidenceReturning(ctx context.Context, projectID, materialID, id, reference string, expiresAt time.Time) (domain.Evidence, error) {
	evidence, err := domain.NewEvidence(id, projectID, materialID, reference, domain.Certificate, "supplier", time.Now())
	if err != nil {
		return domain.Evidence{}, err
	}
	value := expiresAt
	evidence.ExpiresAt = &value
	if err := s.SaveEvidence(ctx, evidence); err != nil {
		return domain.Evidence{}, err
	}
	return evidence, nil
}
