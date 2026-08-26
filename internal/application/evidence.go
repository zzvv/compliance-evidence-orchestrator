package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"time"
)

type RegisterEvidenceCommand struct {
	ProjectID  string
	MaterialID string
	Reference  string
	Kind       domain.EvidenceKind
	Supplier   string
	IssuedAt   time.Time
	ExpiresAt  *time.Time
	Actor      string
}

func (s *EvidenceService) RegisterEvidence(ctx context.Context, command RegisterEvidenceCommand) (domain.Evidence, error) {
	evidence, err := domain.NewEvidence(s.ids.New("ev"), command.ProjectID, command.MaterialID, command.Reference, command.Kind, command.Supplier, command.IssuedAt)
	if err != nil {
		return domain.Evidence{}, err
	}
	evidence.ExpiresAt = command.ExpiresAt
	if err := evidence.Validate(); err != nil {
		return domain.Evidence{}, err
	}
	if err := s.evidence.SaveEvidence(ctx, evidence); err != nil {
		return domain.Evidence{}, err
	}
	s.appendAudit(ctx, evidence.Scope(), "", "evidence_registered", command.Actor)
	return evidence, nil
}
func (s *EvidenceService) EvidenceForScope(ctx context.Context, scope domain.Scope) ([]domain.Evidence, error) {
	return s.evidence.ListEvidence(ctx, scope)
}
