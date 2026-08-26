package domain

import (
	"fmt"
	"path"
	"strings"
	"time"
)

type Attachment struct {
	ID         string    `json:"id"`
	EvidenceID string    `json:"evidence_id"`
	Name       string    `json:"name"`
	MediaType  string    `json:"media_type"`
	SizeBytes  int64     `json:"size_bytes"`
	Digest     string    `json:"digest"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func NewAttachment(id, evidenceID, name, mediaType, digest string, size int64, now time.Time) (Attachment, error) {
	a := Attachment{ID: strings.TrimSpace(id), EvidenceID: strings.TrimSpace(evidenceID), Name: path.Base(strings.TrimSpace(name)), MediaType: strings.TrimSpace(mediaType), SizeBytes: size, Digest: strings.TrimSpace(digest), UploadedAt: now.UTC()}
	if a.ID == "" || a.EvidenceID == "" || a.Name == "." || a.Name == "" {
		return Attachment{}, fmt.Errorf("attachment identity is required")
	}
	if a.SizeBytes <= 0 {
		return Attachment{}, fmt.Errorf("attachment size must be positive")
	}
	if a.Digest == "" {
		return Attachment{}, fmt.Errorf("attachment digest is required")
	}
	return a, nil
}
