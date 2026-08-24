package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"time"
)

func (s *EvidenceService) appendAudit(ctx context.Context, scope domain.Scope, batchID, action, actor string) {
	if s.audits == nil {
		return
	}
	_ = s.audits.AppendAudit(ctx, domain.NewAuditEvent(s.ids.New("audit"), scope, batchID, action, actor, time.Now()))
}
func (s *EvidenceService) AuditTrail(ctx context.Context, scope domain.Scope) ([]domain.AuditEvent, error) {
	if s.audits == nil {
		return []domain.AuditEvent{}, nil
	}
	return s.audits.ListAudit(ctx, scope)
}
