package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"sort"
	"strings"
)

type BatchFilter struct {
	ProjectID  string
	MaterialID string
	State      domain.BatchState
	Limit      int
}

func (s *Store) SearchBatches(ctx context.Context, filter BatchFilter) ([]domain.ReviewBatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]domain.ReviewBatch, 0)
	for _, batch := range s.batches {
		if filter.ProjectID != "" && batch.Scope.ProjectID != filter.ProjectID {
			continue
		}
		if filter.MaterialID != "" && batch.Scope.MaterialID != filter.MaterialID {
			continue
		}
		if filter.State != "" && batch.State != filter.State {
			continue
		}
		values = append(values, copyBatch(batch))
	}
	sort.Slice(values, func(left, right int) bool { return values[left].UpdatedAt.After(values[right].UpdatedAt) })
	if filter.Limit > 0 && len(values) > filter.Limit {
		values = values[:filter.Limit]
	}
	return values, nil
}

func (s *Store) FindEvidenceByReference(ctx context.Context, scope domain.Scope, reference string) ([]domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.TrimSpace(reference)
	values := make([]domain.Evidence, 0)
	for _, evidence := range s.evidence {
		if evidence.Scope() == scope && evidence.Reference == needle {
			values = append(values, evidence)
		}
	}
	sort.Slice(values, func(left, right int) bool { return values[left].CreatedAt.Before(values[right].CreatedAt) })
	return values, nil
}

func (s *Store) CountByState(ctx context.Context, scope domain.Scope) (map[domain.BatchState]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[domain.BatchState]int)
	for _, batch := range s.batches {
		if batch.Scope == scope {
			result[batch.State]++
		}
	}
	return result, nil
}
