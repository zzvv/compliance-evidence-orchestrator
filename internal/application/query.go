package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

type BatchDetail struct {
	Batch    domain.ReviewBatch  `json:"batch"`
	Evidence []domain.Evidence   `json:"evidence"`
	Receipts []domain.Receipt    `json:"receipts"`
	Audit    []domain.AuditEvent `json:"audit"`
}

func (s *EvidenceService) BatchDetail(ctx context.Context, batchID string) (BatchDetail, error) {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return BatchDetail{}, err
	}
	evidence, err := s.resolveEvidence(ctx, batch.Scope, batch.EvidenceIDs)
	if err != nil {
		return BatchDetail{}, err
	}
	receipts, err := s.receipts.ListReceipts(ctx, batchID)
	if err != nil {
		return BatchDetail{}, err
	}
	detail := BatchDetail{Batch: batch, Evidence: evidence, Receipts: receipts}
	if s.audits != nil {
		detail.Audit, _ = s.audits.ListAudit(ctx, batch.Scope)
	}
	return detail, nil
}
func (s *EvidenceService) Batches(ctx context.Context, scope domain.Scope) ([]domain.ReviewBatch, error) {
	return s.batches.ListBatches(ctx, scope)
}
