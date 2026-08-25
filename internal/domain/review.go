package domain

import (
	"fmt"
	"strings"
	"time"
)

type ReviewDecision string

const (
	DecisionApprove ReviewDecision = "approve"
	DecisionReject  ReviewDecision = "reject"
)

type ReviewNote struct {
	ID        string         `json:"id"`
	BatchID   string         `json:"batch_id"`
	Author    string         `json:"author"`
	Decision  ReviewDecision `json:"decision"`
	Text      string         `json:"text"`
	CreatedAt time.Time      `json:"created_at"`
}

func NewReviewNote(id, batchID, author string, decision ReviewDecision, text string, now time.Time) (ReviewNote, error) {
	note := ReviewNote{ID: strings.TrimSpace(id), BatchID: strings.TrimSpace(batchID), Author: strings.TrimSpace(author), Decision: decision, Text: strings.TrimSpace(text), CreatedAt: now.UTC()}
	if note.ID == "" || note.BatchID == "" || note.Author == "" {
		return ReviewNote{}, fmt.Errorf("review note identity is required")
	}
	if note.Decision != DecisionApprove && note.Decision != DecisionReject {
		return ReviewNote{}, fmt.Errorf("invalid decision")
	}
	if note.Decision == DecisionReject && note.Text == "" {
		return ReviewNote{}, fmt.Errorf("rejection note is required")
	}
	return note, nil
}
