package application

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

type fixedEvidenceRepository struct{ values []domain.Evidence }

func (r fixedEvidenceRepository) SaveEvidence(context.Context, domain.Evidence) error { return nil }
func (r fixedEvidenceRepository) ListEvidence(context.Context, domain.Scope) ([]domain.Evidence, error) {
	return append([]domain.Evidence(nil), r.values...), nil
}
func (r fixedEvidenceRepository) FindEvidence(context.Context, domain.Scope, string) (domain.Evidence, error) {
	return domain.Evidence{}, domain.ErrNotFound
}

type changingBatchRepository struct {
	first  []domain.ReviewBatch
	second []domain.ReviewBatch
	calls  int
}

func (r *changingBatchRepository) SaveBatch(context.Context, domain.ReviewBatch) error { return nil }
func (r *changingBatchRepository) FindBatch(context.Context, string) (domain.ReviewBatch, error) {
	return domain.ReviewBatch{}, domain.ErrNotFound
}
func (r *changingBatchRepository) ListBatches(context.Context, domain.Scope) ([]domain.ReviewBatch, error) {
	r.calls++
	if r.calls == 1 {
		return append([]domain.ReviewBatch(nil), r.first...), nil
	}
	return append([]domain.ReviewBatch(nil), r.second...), nil
}

func TestDashboardKeepsSectionsOnOneBatchSnapshot(t *testing.T) {
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := domain.NewBatch("batch-1", scope, []string{"evidence-1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := batch.Transition(domain.BatchSubmitted, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	before := batch
	if err := batch.Transition(domain.BatchUnderReview, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := batch.Transition(domain.BatchApproved, "approved", time.Now()); err != nil {
		t.Fatal(err)
	}

	batches := &changingBatchRepository{first: []domain.ReviewBatch{before}, second: []domain.ReviewBatch{batch}}
	service := NewEvidenceService(fixedEvidenceRepository{}, batches, nil, NewMemoryNotifier())
	dashboard, err := service.Dashboard(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Summary.OpenBatches != dashboard.StateCounts[domain.BatchSubmitted]+dashboard.StateCounts[domain.BatchUnderReview]+dashboard.StateCounts[domain.BatchDraft] {
		t.Fatalf("dashboard combined different batch snapshots: summary=%+v counts=%+v", dashboard.Summary, dashboard.StateCounts)
	}
	if batches.calls != 1 {
		t.Fatalf("dashboard read batches %d times, want one consistent snapshot", batches.calls)
	}
}
