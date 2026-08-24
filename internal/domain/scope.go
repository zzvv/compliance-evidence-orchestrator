package domain

import (
	"fmt"
	"strings"
)

type Scope struct {
	ProjectID  string `json:"project_id"`
	MaterialID string `json:"material_id"`
}

func NewScope(projectID, materialID string) (Scope, error) {
	scope := Scope{ProjectID: strings.TrimSpace(projectID), MaterialID: strings.TrimSpace(materialID)}
	if err := scope.Validate(); err != nil {
		return Scope{}, err
	}
	return scope, nil
}
func (s Scope) Validate() error {
	if s.ProjectID == "" {
		return fmt.Errorf("project id is required")
	}
	if s.MaterialID == "" {
		return fmt.Errorf("material id is required")
	}
	return nil
}
func (s Scope) Key() string { return s.ProjectID + ":" + s.MaterialID }
