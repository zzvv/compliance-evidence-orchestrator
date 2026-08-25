package domain

import "time"

type ScopeSummary struct {
	Scope           Scope     `json:"scope"`
	TotalEvidence   int       `json:"total_evidence"`
	OpenBatches     int       `json:"open_batches"`
	ApprovedBatches int       `json:"approved_batches"`
	RejectedBatches int       `json:"rejected_batches"`
	LastActivityAt  time.Time `json:"last_activity_at"`
}

func SummarizeScope(scope Scope, evidence []Evidence, batches []ReviewBatch) ScopeSummary {
	summary := ScopeSummary{Scope: scope, TotalEvidence: len(evidence)}
	for _, batch := range batches {
		switch batch.State {
		case BatchApproved:
			summary.ApprovedBatches++
		case BatchRejected:
			summary.RejectedBatches++
		case BatchCancelled:
		default:
			summary.OpenBatches++
		}
		if batch.UpdatedAt.After(summary.LastActivityAt) {
			summary.LastActivityAt = batch.UpdatedAt
		}
	}
	return summary
}
func (s ScopeSummary) HasOpenReview() bool { return s.OpenBatches > 0 }
