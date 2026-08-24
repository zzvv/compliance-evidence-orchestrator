package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ChecklistItemState string

const (
	ChecklistPending   ChecklistItemState = "pending"
	ChecklistSatisfied ChecklistItemState = "satisfied"
	ChecklistWaived    ChecklistItemState = "waived"
)

type ChecklistItem struct {
	Code       string             `json:"code"`
	Label      string             `json:"label"`
	State      ChecklistItemState `json:"state"`
	EvidenceID string             `json:"evidence_id,omitempty"`
	Note       string             `json:"note,omitempty"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type Checklist struct {
	BatchID   string          `json:"batch_id"`
	Items     []ChecklistItem `json:"items"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func NewChecklist(batchID string, now time.Time) Checklist {
	items := []ChecklistItem{
		{Code: "identity", Label: "材料与项目范围一致", State: ChecklistPending, UpdatedAt: now.UTC()},
		{Code: "currency", Label: "证据在有效期内", State: ChecklistPending, UpdatedAt: now.UTC()},
		{Code: "certificate", Label: "已附合规证书", State: ChecklistPending, UpdatedAt: now.UTC()},
	}
	return Checklist{BatchID: strings.TrimSpace(batchID), Items: items, UpdatedAt: now.UTC()}
}

func (c Checklist) Validate() error {
	if c.BatchID == "" {
		return fmt.Errorf("checklist batch id is required")
	}
	if len(c.Items) == 0 {
		return fmt.Errorf("checklist needs items")
	}
	seen := make(map[string]struct{}, len(c.Items))
	for _, item := range c.Items {
		if item.Code == "" || item.Label == "" {
			return fmt.Errorf("checklist item identity is required")
		}
		if _, ok := seen[item.Code]; ok {
			return fmt.Errorf("duplicate checklist item %q", item.Code)
		}
		seen[item.Code] = struct{}{}
		switch item.State {
		case ChecklistPending, ChecklistSatisfied, ChecklistWaived:
		default:
			return fmt.Errorf("invalid checklist state")
		}
	}
	return nil
}

func (c *Checklist) Mark(code string, state ChecklistItemState, evidenceID, note string, now time.Time) error {
	for index := range c.Items {
		if c.Items[index].Code != code {
			continue
		}
		if state == ChecklistSatisfied && strings.TrimSpace(evidenceID) == "" {
			return fmt.Errorf("satisfied checklist item requires evidence")
		}
		c.Items[index].State = state
		c.Items[index].EvidenceID = strings.TrimSpace(evidenceID)
		c.Items[index].Note = strings.TrimSpace(note)
		c.Items[index].UpdatedAt = now.UTC()
		c.UpdatedAt = now.UTC()
		return nil
	}
	return fmt.Errorf("checklist item %q not found", code)
}

func (c Checklist) Complete() bool {
	for _, item := range c.Items {
		if item.State == ChecklistPending {
			return false
		}
	}
	return true
}
func (c Checklist) PendingCodes() []string {
	values := make([]string, 0)
	for _, item := range c.Items {
		if item.State == ChecklistPending {
			values = append(values, item.Code)
		}
	}
	sort.Strings(values)
	return values
}
