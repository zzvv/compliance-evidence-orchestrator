package domain

import (
	"fmt"
	"time"
)

type ReviewPolicy struct {
	RequireCurrentEvidence bool
	RequireCertificate     bool
	MaxEvidencePerBatch    int
}

func DefaultReviewPolicy() ReviewPolicy {
	return ReviewPolicy{RequireCurrentEvidence: true, RequireCertificate: true, MaxEvidencePerBatch: 12}
}
func (p ReviewPolicy) Validate(evidence []Evidence) error {
	if len(evidence) == 0 {
		return fmt.Errorf("evidence is required")
	}
	if p.MaxEvidencePerBatch > 0 && len(evidence) > p.MaxEvidencePerBatch {
		return fmt.Errorf("batch exceeds evidence limit")
	}
	certificate := false
	for _, item := range evidence {
		if p.RequireCurrentEvidence && item.IsExpired(time.Now()) {
			return fmt.Errorf("evidence %s is expired", item.ID)
		}
		if item.Kind == Certificate {
			certificate = true
		}
	}
	if p.RequireCertificate && !certificate {
		return fmt.Errorf("a certificate is required")
	}
	return nil
}
