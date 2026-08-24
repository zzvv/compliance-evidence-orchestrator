package domain

import "strings"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type RiskAssessment struct {
	Level   RiskLevel `json:"level"`
	Reasons []string  `json:"reasons"`
}

func AssessRisk(evidence []Evidence, batch ReviewBatch) RiskAssessment {
	reasons := make([]string, 0)
	certificate := false
	for _, item := range evidence {
		if item.Kind == Certificate {
			certificate = true
		}
		if item.IsExpired(batch.UpdatedAt) {
			reasons = append(reasons, "存在过期材料")
		}
		if strings.TrimSpace(item.Supplier) == "" {
			reasons = append(reasons, "供应商信息缺失")
		}
	}
	if !certificate {
		reasons = append(reasons, "没有合规证书")
	}
	level := RiskLow
	if len(reasons) > 0 {
		level = RiskMedium
	}
	if len(reasons) > 1 || batch.State == BatchRejected {
		level = RiskHigh
	}
	return RiskAssessment{Level: level, Reasons: reasons}
}
