package application

import (
	"context"
	"fmt"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// OperationResult is used by callers that need to present a stable workflow
// message while retaining the batch identifier for later inspection.
type OperationResult struct {
	BatchID string            `json:"batch_id"`
	State   domain.BatchState `json:"state"`
	Message string            `json:"message"`
	At      time.Time         `json:"at"`
}

func (s *EvidenceService) SubmitAndDescribe(ctx context.Context, command CreateBatchCommand) (OperationResult, error) {
	batch, err := s.CreateBatch(ctx, command)
	if err != nil {
		return OperationResult{}, err
	}
	return OperationResult{BatchID: batch.ID, State: batch.State, Message: "材料已进入审核队列", At: batch.UpdatedAt}, nil
}

func (s *EvidenceService) DescribeBatch(ctx context.Context, batchID string) (OperationResult, error) {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return OperationResult{}, err
	}
	message := stateMessage(batch.State)
	if batch.Reason != "" {
		message = fmt.Sprintf("%s：%s", message, batch.Reason)
	}
	return OperationResult{BatchID: batch.ID, State: batch.State, Message: message, At: batch.UpdatedAt}, nil
}

func stateMessage(state domain.BatchState) string {
	switch state {
	case domain.BatchDraft:
		return "批次尚未提交"
	case domain.BatchSubmitted:
		return "等待审核"
	case domain.BatchUnderReview:
		return "审核进行中"
	case domain.BatchApproved:
		return "审核已通过"
	case domain.BatchRejected:
		return "审核未通过"
	case domain.BatchCancelled:
		return "审核已取消"
	default:
		return "未知审核状态"
	}
}

func (s *EvidenceService) IsVisibleToReviewer(ctx context.Context, batchID string) (bool, error) {
	batch, err := s.batches.FindBatch(ctx, batchID)
	if err != nil {
		return false, err
	}
	return batch.State == domain.BatchSubmitted || batch.State == domain.BatchUnderReview, nil
}
