package application

import (
	"context"
	"fmt"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func (s *EvidenceService) ValidateBatch(ctx context.Context, batchID string) error {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return err
	}
	evidence, err := s.resolveEvidence(ctx, batch.Scope, batch.EvidenceIDs)
	if err != nil {
		return err
	}
	if err := s.policy.Validate(evidence); err != nil {
		return fmt.Errorf("batch %s no longer satisfies review policy: %w", batch.ID, err)
	}
	return nil
}
func EvidenceBelongsToScope(evidence domain.Evidence, scope domain.Scope) bool {
	return evidence.ProjectID == scope.ProjectID && evidence.MaterialID == scope.MaterialID
}
