package transport

import (
	"github.com/zzvv/compliance-evidence-orchestrator/internal/application"
	"net/http"
)

func NewRouter(service *application.EvidenceService) http.Handler {
	handler := NewHandler(service)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.health)
	mux.HandleFunc("POST /v1/evidence", handler.registerEvidence)
	mux.HandleFunc("POST /v1/batches", handler.createBatch)
	mux.HandleFunc("GET /v1/batches/{id}", handler.batchDetail)
	mux.HandleFunc("POST /v1/batches/{id}/start", handler.startReview)
	mux.HandleFunc("POST /v1/batches/{id}/decision", handler.decideBatch)
	mux.HandleFunc("POST /v1/batches/{id}/cancel", handler.cancelBatch)
	mux.HandleFunc("GET /v1/scopes/{project}/{material}/summary", handler.scopeSummary)
	mux.HandleFunc("GET /v1/scopes/{project}/{material}/work-plan", handler.workPlan)
	mux.HandleFunc("GET /", handler.console)
	return logging(mux)
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
