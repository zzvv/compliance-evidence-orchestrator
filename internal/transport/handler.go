package transport

import (
	"encoding/json"
	"errors"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/application"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"net/http"
	"time"
)

type Handler struct{ service *application.EvidenceService }

func NewHandler(service *application.EvidenceService) *Handler { return &Handler{service: service} }
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) registerEvidence(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ProjectID  string              `json:"project_id"`
		MaterialID string              `json:"material_id"`
		Reference  string              `json:"reference"`
		Kind       domain.EvidenceKind `json:"kind"`
		Supplier   string              `json:"supplier"`
		IssuedAt   time.Time           `json:"issued_at"`
		ExpiresAt  *time.Time          `json:"expires_at"`
		Actor      string              `json:"actor"`
	}
	if !decode(w, r, &request) {
		return
	}
	item, err := h.service.RegisterEvidence(r.Context(), application.RegisterEvidenceCommand{ProjectID: request.ProjectID, MaterialID: request.MaterialID, Reference: request.Reference, Kind: request.Kind, Supplier: request.Supplier, IssuedAt: request.IssuedAt, ExpiresAt: request.ExpiresAt, Actor: request.Actor})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (h *Handler) createBatch(w http.ResponseWriter, r *http.Request) {
	var request application.CreateBatchCommand
	if !decode(w, r, &request) {
		return
	}
	batch, err := h.service.CreateBatch(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, batch)
}
func (h *Handler) batchDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.BatchDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
func (h *Handler) startReview(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Actor string `json:"actor"`
	}
	if !decode(w, r, &request) {
		return
	}
	batch, err := h.service.StartReview(r.Context(), r.PathValue("id"), request.Actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
func (h *Handler) decideBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Approved  bool   `json:"approved"`
		Reason    string `json:"reason"`
		Actor     string `json:"actor"`
		Recipient string `json:"recipient"`
	}
	if !decode(w, r, &request) {
		return
	}
	batch, err := h.service.DecideBatch(r.Context(), application.DecideBatchCommand{BatchID: r.PathValue("id"), Approved: request.Approved, Reason: request.Reason, Actor: request.Actor, Recipient: request.Recipient})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
func (h *Handler) cancelBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
		Actor  string `json:"actor"`
	}
	if !decode(w, r, &request) {
		return
	}
	batch, err := h.service.CancelBatch(r.Context(), r.PathValue("id"), request.Reason, request.Actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, batch)
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body is required"})
		return false
	}
	defer r.Body.Close()
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, domain.ErrNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrInvalidState) {
		status = http.StatusConflict
	}
	if errors.Is(err, contextCanceled()) {
		status = 499
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func contextCanceled() error { return domain.ErrCancelled }
