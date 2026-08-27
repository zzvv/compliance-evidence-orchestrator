package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

type WorkPlanOptions struct {
	Now           time.Time
	IncludeClosed bool
	Limit         int
}

func (s *EvidenceService) WorkPlan(ctx context.Context, scope domain.Scope, options WorkPlanOptions) (domain.WorkPlan, error) {
	if err := scope.Validate(); err != nil {
		return domain.WorkPlan{}, err
	}
	evidence, err := s.evidence.ListEvidence(ctx, scope)
	if err != nil {
		return domain.WorkPlan{}, err
	}
	batches, err := s.batches.ListBatches(ctx, scope)
	if err != nil {
		return domain.WorkPlan{}, err
	}
	if !options.IncludeClosed {
		batches = actionableBatches(batches)
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	plan, err := domain.BuildWorkPlan(scope, evidence, batches, now)
	if err != nil {
		return domain.WorkPlan{}, err
	}
	if options.Limit > 0 && len(plan.Actions) > options.Limit {
		plan.Actions = append([]domain.WorkAction(nil), plan.Actions[:options.Limit]...)
	}
	return plan, nil
}

func (s *EvidenceService) NextWorkAction(ctx context.Context, projectID, materialID string) (domain.WorkAction, error) {
	scope, err := domain.NewScope(projectID, materialID)
	if err != nil {
		return domain.WorkAction{}, err
	}
	plan, err := s.WorkPlan(ctx, scope, WorkPlanOptions{Limit: 1})
	if err != nil {
		return domain.WorkAction{}, err
	}
	action, ok := plan.NextAction()
	if !ok {
		return domain.WorkAction{}, fmt.Errorf("no pending work for %s", scope.Key())
	}
	return action, nil
}

func actionableBatches(batches []domain.ReviewBatch) []domain.ReviewBatch {
	result := make([]domain.ReviewBatch, 0, len(batches))
	for _, batch := range batches {
		if batch.State == domain.BatchApproved || batch.State == domain.BatchCancelled {
			continue
		}
		result = append(result, batch)
	}
	return result
}

func ParseWorkPlanOptions(value string) (WorkPlanOptions, error) {
	options := WorkPlanOptions{}
	for _, item := range strings.Split(strings.TrimSpace(value), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		switch item {
		case "include_closed":
			options.IncludeClosed = true
		default:
			return WorkPlanOptions{}, fmt.Errorf("unsupported work plan option %q", item)
		}
	}
	return options, nil
}
