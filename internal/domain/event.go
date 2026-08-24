package domain

import "time"

type AuditEvent struct {
	ID         string            `json:"id"`
	Scope      Scope             `json:"scope"`
	BatchID    string            `json:"batch_id"`
	Action     string            `json:"action"`
	Actor      string            `json:"actor"`
	At         time.Time         `json:"at"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

func NewAuditEvent(id string, scope Scope, batchID, action, actor string, at time.Time) AuditEvent {
	return AuditEvent{ID: id, Scope: scope, BatchID: batchID, Action: action, Actor: actor, At: at.UTC(), Attributes: map[string]string{}}
}
