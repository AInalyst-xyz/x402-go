package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/x402-rs/x402-go/pkg/facilitator"
	"github.com/x402-rs/x402-go/pkg/types"
)

// Handler manages HTTP handlers for the facilitator
type Handler struct {
	facilitator facilitator.Facilitator
}

// NewHandler creates a new HTTP handler
func NewHandler(fac facilitator.Facilitator) *Handler {
	return &Handler{
		facilitator: fac,
	}
}

// VerifyHandler handles POST /verify requests
func (h *Handler) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request (fail on unknown/misnamed fields)
	var req types.VerifyRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Verify payment
	resp, err := h.facilitator.Verify(r.Context(), &req)
	if err != nil {
		// Protocol-level errors return 200 with invalid response
		if facErr, ok := err.(*types.FacilitatorError); ok {
			respondJSON(w, http.StatusOK, types.NewInvalidResponse(facErr.Message, facErr.Payer))
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("verification failed: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// SettleHandler handles POST /settle requests
func (h *Handler) SettleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req types.SettleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Settle payment
	resp, err := h.facilitator.Settle(r.Context(), &req)
	if err != nil {
		// Protocol-level errors return 200 with error in response
		if facErr, ok := err.(*types.FacilitatorError); ok {
			respondJSON(w, http.StatusOK, types.SettleResponse{
				Success: false,
				Error:   facErr.Message,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("settlement failed: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// SupportedHandler handles GET /supported requests
func (h *Handler) SupportedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := h.facilitator.Supported(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get supported kinds: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// HealthHandler handles GET /health requests
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// SetupRoutes sets up all HTTP routes
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/verify", h.VerifyHandler)
	mux.HandleFunc("/settle", h.SettleHandler)
	mux.HandleFunc("/supported", h.SupportedHandler)
	mux.HandleFunc("/health", h.HealthHandler)
}
