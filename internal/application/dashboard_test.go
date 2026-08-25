package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

// openStates are the batch states the summary counts as "open". The dashboard's
// summary, state counts and recent list must all agree on these counts since
// they are now derived from one consistent snapshot.
var openStates = []domain.BatchState{domain.BatchDraft, domain.BatchSubmitted, domain.BatchUnderReview}

func seedBatch(t *testing.T, store *repository.Store, scope domain.Scope, id string, state domain.BatchState, updatedAt time.Time) domain.ReviewBatch {
	t.Helper()
	batch, err := domain.NewBatch(id, scope, []string{"ev-" + id}, time.Now())
	if err != nil {
		t.Fatalf("new batch %s: %v", id, err)
	}
	batch.State = state
	batch.UpdatedAt = updatedAt
	if err := store.SaveBatch(context.Background(), batch); err != nil {
		t.Fatalf("save batch %s: %v", id, err)
	}
	return batch
}

func dashboardOpenCount(counts map[domain.BatchState]int) int {
	open := 0
	for _, state := range openStates {
		open += counts[state]
	}
	return open
}

// TestDashboardSectionsAreConsistent exercises the fix: the summary's open
// count, the per-state counts and the recent list must all be sourced from one
// snapshot. Before the fix the summary came from one ListBatches call while the
// counts and recent list came from another, so a concurrent transition could
// leave them disagreeing (e.g. summary shows one open while counts show none).
func TestDashboardSectionsAreConsistent(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC()
	seedBatch(t, store, scope, "b-draft", domain.BatchDraft, base.Add(1*time.Second))
	seedBatch(t, store, scope, "b-submitted", domain.BatchSubmitted, base.Add(2*time.Second))
	seedBatch(t, store, scope, "b-review", domain.BatchUnderReview, base.Add(3*time.Second))
	seedBatch(t, store, scope, "b-approved", domain.BatchApproved, base.Add(4*time.Second))
	seedBatch(t, store, scope, "b-rejected", domain.BatchRejected, base.Add(5*time.Second))
	seedBatch(t, store, scope, "b-cancelled", domain.BatchCancelled, base.Add(6*time.Second))

	dash, err := service.Dashboard(ctx, scope)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if got, want := dashboardOpenCount(dash.StateCounts), dash.Summary.OpenBatches; got != want {
		t.Fatalf("state counts open %d != summary open %d (sections disagree on the same snapshot)", got, want)
	}
	if got, want := dash.Summary.ApprovedBatches, dash.StateCounts[domain.BatchApproved]; got != want {
		t.Fatalf("summary approved %d != state count %d", got, want)
	}
	if got, want := dash.Summary.RejectedBatches, dash.StateCounts[domain.BatchRejected]; got != want {
		t.Fatalf("summary rejected %d != state count %d", got, want)
	}
	// The recent list is a window over the same batches, so every entry must be
	// present in the per-state counts and contribute to its state's total.
	totalInStateCounts := 0
	for _, count := range dash.StateCounts {
		totalInStateCounts += count
	}
	if got, want := totalInStateCounts, 6; got != want {
		t.Fatalf("state counts cover %d batches, want %d", got, want)
	}
	for _, batch := range dash.Recent {
		if dash.StateCounts[batch.State] <= 0 {
			t.Fatalf("recent batch %s in state %s not represented in state counts", batch.ID, batch.State)
		}
	}
}

// TestDashboardRecentSortedAndCapped verifies the dashboard keeps its existing
// sorting (UpdatedAt descending) and 10-entry cap over the snapshot.
func TestDashboardRecentSortedAndCapped(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	scope, err := domain.NewScope("project-a", "material-b")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC()
	for i := 0; i < 12; i++ {
		seedBatch(t, store, scope, "b-"+string(rune('a'+i)), domain.BatchSubmitted, base.Add(time.Duration(i)*time.Second))
	}
	dash, err := service.Dashboard(ctx, scope)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if len(dash.Recent) != 10 {
		t.Fatalf("recent length = %d, want 10", len(dash.Recent))
	}
	for i := 1; i < len(dash.Recent); i++ {
		if dash.Recent[i-1].UpdatedAt.Before(dash.Recent[i].UpdatedAt) {
			t.Fatalf("recent not sorted by UpdatedAt descending at index %d", i)
		}
	}
	if dash.Recent[0].UpdatedAt != base.Add(11*time.Second) {
		t.Fatalf("recent head UpdatedAt = %v, want newest", dash.Recent[0].UpdatedAt)
	}
	// State counts must still reflect all 12 batches, not just the 10 shown.
	if got := dashboardOpenCount(dash.StateCounts); got != 12 {
		t.Fatalf("state counts open = %d, want 12 (counts cover the full snapshot)", got)
	}
}

// TestDashboardIsScopedToMaterial confirms batches in another material scope
// never leak into this dashboard.
func TestDashboardIsScopedToMaterial(t *testing.T) {
	store := repository.NewStore()
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	other, err := domain.NewScope("project-a", "material-b")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC()
	seedBatch(t, store, scope, "b-own", domain.BatchSubmitted, base)
	seedBatch(t, store, other, "b-other", domain.BatchSubmitted, base)

	dash, err := service.Dashboard(ctx, scope)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if len(dash.Recent) != 1 || dash.Recent[0].ID != "b-own" {
		t.Fatalf("recent = %+v, want only b-own", dash.Recent)
	}
	if dashboardOpenCount(dash.StateCounts) != 1 {
		t.Fatalf("state counts leaked across scopes: %+v", dash.StateCounts)
	}
}

// TestDashboardScopeViewReaderUsed drives the consistent-snapshot path
// directly: a ScopeViewReader that hands back a fixed view lets us assert the
// dashboard builds all three sections from that single view without any extra
// repository reads.
func TestDashboardScopeViewReaderUsed(t *testing.T) {
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := domain.NewEvidence("ev-1", "project-a", "material-a", "CERT-1", domain.Certificate, "supplier", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	batch, err := domain.NewBatch("b-1", scope, []string{"ev-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	batch.State = domain.BatchUnderReview
	reader := &stubScopeViewReader{
		view: repository.ScopeView{Evidence: []domain.Evidence{evidence}, Batches: []domain.ReviewBatch{batch}},
	}
	// Wire the service with nil repositories but the scope-view reader set, the
	// path the real *Store exercises in production.
	service := &EvidenceService{scopeView: reader, policy: domain.DefaultReviewPolicy()}
	dash, err := service.Dashboard(context.Background(), scope)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("scope view reader called %d times, want exactly 1 (one consistent snapshot)", reader.calls)
	}
	if dash.Summary.OpenBatches != 1 || dash.StateCounts[domain.BatchUnderReview] != 1 || len(dash.Recent) != 1 {
		t.Fatalf("dashboard sections not built from the snapshot: %+v", dash)
	}
}

type stubScopeViewReader struct {
	view  repository.ScopeView
	calls int
}

func (s *stubScopeViewReader) ReadScopeView(_ context.Context, _ domain.Scope) (repository.ScopeView, error) {
	s.calls++
	return s.view, nil
}
