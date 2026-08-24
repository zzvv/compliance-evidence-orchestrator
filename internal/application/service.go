package application

import (
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

type EvidenceService struct {
	evidence      repository.EvidenceRepository
	batches       repository.BatchRepository
	receipts      repository.ReceiptRepository
	notifications repository.NotificationRepository
	audits        repository.AuditRepository
	notifier      Notifier
	ids           IDGenerator
	policy        domain.ReviewPolicy
}

func NewEvidenceService(evidence repository.EvidenceRepository, batches repository.BatchRepository, receipts repository.ReceiptRepository, notifier Notifier) *EvidenceService {
	service := &EvidenceService{evidence: evidence, batches: batches, receipts: receipts, notifier: notifier, policy: domain.DefaultReviewPolicy()}
	if n, ok := evidence.(repository.NotificationRepository); ok {
		service.notifications = n
	}
	if a, ok := evidence.(repository.AuditRepository); ok {
		service.audits = a
	}
	return service
}
func (s *EvidenceService) WithPolicy(policy domain.ReviewPolicy) *EvidenceService {
	s.policy = policy
	return s
}
