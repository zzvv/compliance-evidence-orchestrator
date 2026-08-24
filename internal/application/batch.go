package application

import (
	"context"
	"fmt"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"time"
)

type CreateBatchCommand struct {
	ProjectID   string
	MaterialID  string
	EvidenceIDs []string
	Actor       string
}
type DecideBatchCommand struct {
	BatchID   string
	Approved  bool
	Reason    string
	Actor     string
	Recipient string
}

func (s *EvidenceService) CreateBatch(ctx context.Context, command CreateBatchCommand) (domain.ReviewBatch, error) {
	scope, err := domain.NewScope(command.ProjectID, command.MaterialID)
	if err != nil {
		return domain.ReviewBatch{}, err
	}
	items, err := s.resolveEvidence(ctx, scope, command.EvidenceIDs)
	if err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.policy.Validate(items); err != nil {
		return domain.ReviewBatch{}, err
	}
	now := time.Now().UTC()
	batch, err := domain.NewBatch(s.ids.New("batch"), scope, command.EvidenceIDs, now)
	if err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := batch.Transition(domain.BatchSubmitted, "", now); err != nil {
		return domain.ReviewBatch{}, err
	}
	receipt := domain.NewReceipt(s.ids.New("receipt"), batch.ID, domain.ReceiptSubmitted, "materials submitted for review", now)
	if err := s.batches.SaveBatch(ctx, batch); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.receipts.AppendReceipt(ctx, receipt); err != nil {
		return domain.ReviewBatch{}, err
	}
	s.appendAudit(ctx, scope, batch.ID, "batch_submitted", command.Actor)
	return batch, nil
}
func (s *EvidenceService) StartReview(ctx context.Context, batchID, actor string) (domain.ReviewBatch, error) {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := batch.Transition(domain.BatchUnderReview, "", time.Now()); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.batches.SaveBatch(ctx, batch); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.receipts.AppendReceipt(ctx, domain.NewReceipt(s.ids.New("receipt"), batch.ID, domain.ReceiptReviewStarted, "review started", time.Now())); err != nil {
		return domain.ReviewBatch{}, err
	}
	s.appendAudit(ctx, batch.Scope, batch.ID, "review_started", actor)
	return batch, nil
}
func (s *EvidenceService) DecideBatch(ctx context.Context, command DecideBatchCommand) (domain.ReviewBatch, error) {
	batch, err := s.batches.FindBatch(ctx, command.BatchID)
	if err != nil {
		return domain.ReviewBatch{}, err
	}
	target := domain.BatchRejected
	kind := domain.ReceiptRejected
	event := "review_rejected"
	if command.Approved {
		target = domain.BatchApproved
		kind = domain.ReceiptApproved
		event = "review_approved"
	}
	if err := batch.Transition(target, command.Reason, time.Now()); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.batches.SaveBatch(ctx, batch); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.receipts.AppendReceipt(ctx, domain.NewReceipt(s.ids.New("receipt"), batch.ID, kind, fmt.Sprintf("review %s", target), time.Now())); err != nil {
		return domain.ReviewBatch{}, err
	}
	s.appendAudit(ctx, batch.Scope, batch.ID, event, command.Actor)
	if command.Recipient != "" {
		s.queueNotification(ctx, batch, command.Recipient, event)
	}
	return batch, nil
}
func (s *EvidenceService) CancelBatch(ctx context.Context, batchID, reason, actor string) (domain.ReviewBatch, error) {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := batch.Transition(domain.BatchCancelled, reason, time.Now()); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.batches.SaveBatch(ctx, batch); err != nil {
		return domain.ReviewBatch{}, err
	}
	if err := s.receipts.AppendReceipt(ctx, domain.NewReceipt(s.ids.New("receipt"), batch.ID, domain.ReceiptCancelled, "review cancelled", time.Now())); err != nil {
		return domain.ReviewBatch{}, err
	}
	s.appendAudit(ctx, batch.Scope, batch.ID, "batch_cancelled", actor)
	return batch, nil
}
func (s *EvidenceService) resolveEvidence(ctx context.Context, scope domain.Scope, ids []string) ([]domain.Evidence, error) {
	items := make([]domain.Evidence, 0, len(ids))
	for _, id := range ids {
		item, err := s.evidence.FindEvidence(ctx, scope, id)
		if err != nil {
			return nil, fmt.Errorf("evidence %s: %w", id, err)
		}
		items = append(items, item)
	}
	return items, nil
}
