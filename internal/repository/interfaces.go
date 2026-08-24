package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

type EvidenceRepository interface {
	SaveEvidence(context.Context, domain.Evidence) error
	ListEvidence(context.Context, domain.Scope) ([]domain.Evidence, error)
	FindEvidence(context.Context, domain.Scope, string) (domain.Evidence, error)
}
type BatchRepository interface {
	SaveBatch(context.Context, domain.ReviewBatch) error
	FindBatch(context.Context, string) (domain.ReviewBatch, error)
	ListBatches(context.Context, domain.Scope) ([]domain.ReviewBatch, error)
}
type ReceiptRepository interface {
	AppendReceipt(context.Context, domain.Receipt) error
	ListReceipts(context.Context, string) ([]domain.Receipt, error)
}
type NotificationRepository interface {
	SaveNotification(context.Context, domain.Notification) error
	PendingNotifications(context.Context, int) ([]domain.Notification, error)
}
type AuditRepository interface {
	AppendAudit(context.Context, domain.AuditEvent) error
	ListAudit(context.Context, domain.Scope) ([]domain.AuditEvent, error)
}
