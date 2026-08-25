package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"sort"
)

type Dashboard struct {
	Scope       domain.Scope              `json:"scope"`
	Summary     domain.ScopeSummary       `json:"summary"`
	StateCounts map[domain.BatchState]int `json:"state_counts"`
	Recent      []domain.ReviewBatch      `json:"recent"`
}

func (s *EvidenceService) Dashboard(ctx context.Context, scope domain.Scope) (Dashboard, error) {
	summary, err := s.ScopeSummary(ctx, scope)
	if err != nil {
		return Dashboard{}, err
	}
	batches, err := s.batches.ListBatches(ctx, scope)
	if err != nil {
		return Dashboard{}, err
	}
	counts := make(map[domain.BatchState]int)
	for _, batch := range batches {
		counts[batch.State]++
	}
	sort.Slice(batches, func(left, right int) bool { return batches[left].UpdatedAt.After(batches[right].UpdatedAt) })
	if len(batches) > 10 {
		batches = batches[:10]
	}
	return Dashboard{Scope: scope, Summary: summary, StateCounts: counts, Recent: batches}, nil
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
