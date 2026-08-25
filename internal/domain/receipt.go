package domain

import "time"

type ReceiptKind string

const (
	ReceiptSubmitted     ReceiptKind = "submitted"
	ReceiptReviewStarted ReceiptKind = "review_started"
	ReceiptApproved      ReceiptKind = "approved"
	ReceiptRejected      ReceiptKind = "rejected"
	ReceiptCancelled     ReceiptKind = "cancelled"
)

type Receipt struct {
	ID         string      `json:"id"`
	BatchID    string      `json:"batch_id"`
	Kind       ReceiptKind `json:"kind"`
	Message    string      `json:"message"`
	RecordedAt time.Time   `json:"recorded_at"`
}

func NewReceipt(id, batchID string, kind ReceiptKind, message string, now time.Time) Receipt {
	return Receipt{ID: id, BatchID: batchID, Kind: kind, Message: message, RecordedAt: now.UTC()}
}
