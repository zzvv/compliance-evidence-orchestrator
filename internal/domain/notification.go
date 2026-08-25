package domain

import "time"

type NotificationState string

const (
	NotificationPending   NotificationState = "pending"
	NotificationDelivered NotificationState = "delivered"
	NotificationFailed    NotificationState = "failed"
)

type Notification struct {
	ID        string            `json:"id"`
	BatchID   string            `json:"batch_id"`
	Recipient string            `json:"recipient"`
	Event     string            `json:"event"`
	State     NotificationState `json:"state"`
	Attempts  int               `json:"attempts"`
	LastError string            `json:"last_error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (n *Notification) MarkDelivered(now time.Time) {
	n.State = NotificationDelivered
	n.Attempts++
	n.LastError = ""
	n.UpdatedAt = now.UTC()
}
func (n *Notification) MarkFailed(err error, now time.Time) {
	n.State = NotificationFailed
	n.Attempts++
	if err != nil {
		n.LastError = err.Error()
	}
	n.UpdatedAt = now.UTC()
}
