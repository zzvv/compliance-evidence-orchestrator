package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
	"sort"
)

type Dashboard struct {
	Scope       domain.Scope              `json:"scope"`
	Summary     domain.ScopeSummary       `json:"summary"`
	StateCounts map[domain.BatchState]int `json:"state_counts"`
	Recent      []domain.ReviewBatch      `json:"recent"`
}

func (s *EvidenceService) Dashboard(ctx context.Context, scope domain.Scope) (Dashboard, error) {
	view, err := s.readScopeView(ctx, scope)
	if err != nil {
		return Dashboard{}, err
	}
	summary := domain.SummarizeScope(scope, view.Evidence, view.Batches)
	counts := make(map[domain.BatchState]int)
	for _, batch := range view.Batches {
		counts[batch.State]++
	}
	recent := append([]domain.ReviewBatch(nil), view.Batches...)
	sort.Slice(recent, func(left, right int) bool { return recent[left].UpdatedAt.After(recent[right].UpdatedAt) })
	if len(recent) > 10 {
		recent = recent[:10]
	}
	return Dashboard{Scope: scope, Summary: summary, StateCounts: counts, Recent: recent}, nil
}

// readScopeView returns the evidence and batches for scope from a single
// consistent snapshot so the dashboard never blends values observed across
// concurrent writes. When the backing repository exposes ScopeViewReader, the
// snapshot is taken under one read lock; otherwise it falls back to the
// standalone reads, which still keeps the dashboard's three sections sourced
// from one batch set.
func (s *EvidenceService) readScopeView(ctx context.Context, scope domain.Scope) (repository.ScopeView, error) {
	if s.scopeView != nil {
		return s.scopeView.ReadScopeView(ctx, scope)
	}
	evidence, err := s.evidence.ListEvidence(ctx, scope)
	if err != nil {
		return repository.ScopeView{}, err
	}
	batches, err := s.batches.ListBatches(ctx, scope)
	if err != nil {
		return repository.ScopeView{}, err
	}
	return repository.ScopeView{Evidence: evidence, Batches: batches}, nil
}
func (s *EvidenceService) RiskForBatch(ctx context.Context, batchID string) (domain.RiskAssessment, error) {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return domain.RiskAssessment{}, err
	}
	evidence, err := s.resolveEvidence(ctx, batch.Scope, batch.EvidenceIDs)
	if err != nil {
		return domain.RiskAssessment{}, err
	}
	return domain.AssessRisk(evidence, batch), nil
}
