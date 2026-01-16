package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type HTTPHandler struct {
	logger *slog.Logger
}

func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{
		logger: slog.Default(),
	}
}

func (h *HTTPHandler) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/healthz", h.HealthHandler)

	return r
}

func (h *HTTPHandler) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	data, err := json.Marshal(HealthResponse{Status: HealthStatusOk})
	if err != nil {
		h.logger.Error("", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
