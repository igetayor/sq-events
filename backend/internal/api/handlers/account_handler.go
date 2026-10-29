// Package handlers contains all HTTP handler functions (controllers) for the
// Event Sourcing REST API. Each handler is a thin adapter: parse request,
// delegate to the domain layer, and write a typed JSON response.
//
// Routes registered by the server bootstrap (cmd/server/main.go):
//   GET  /api/account   → GetAccount
//   POST /api/events    → AppendEvent
//   POST /api/reset     → Reset
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ige-sqlite-event-sourcing/backend/internal/domain/eventstore"
)

// CommandRequest is the inbound DTO for the POST /api/events endpoint.
type CommandRequest struct {
	StreamID        string  `json:"stream_id"`
	EventType       string  `json:"event_type"` // ACCOUNT_OPENED | MONEY_DEPOSITED | MONEY_WITHDRAWN | ACCOUNT_FROZEN | ACCOUNT_UNFROZEN
	Owner           string  `json:"owner"`
	Amount          float64 `json:"amount"`
	Reason          string  `json:"reason"`
	ExpectedVersion int64   `json:"expected_version"`
}

// Handler holds the dependencies injected from the server bootstrap.
type Handler struct {
	store *eventstore.FileEventStore
}

// New creates a Handler with its required dependencies.
func New(store *eventstore.FileEventStore) *Handler {
	return &Handler{store: store}
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// GetAccount rehydrates the aggregate state from the full event stream.
//
//	GET /api/account?stream_id=<id>
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	streamID := r.URL.Query().Get("stream_id")
	if streamID == "" {
		streamID = "acc_prime_001"
	}

	events, err := h.store.LoadStream(streamID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Seed initial state if stream is empty
	if len(events) == 0 {
		_, _ = h.store.Append(streamID, "ACCOUNT_OPENED", map[string]interface{}{
			"account_id":      streamID,
			"owner":           "Ige Institutional Vault",
			"initial_balance": 10000.00,
		}, -1)
		_, _ = h.store.Append(streamID, "MONEY_DEPOSITED", map[string]interface{}{
			"amount": 2500.00,
			"reason": "Initial Treasury Allocation",
		}, -1)
		events, _ = h.store.LoadStream(streamID)
	}

	state, err := eventstore.RehydrateAccount(events)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":  state,
		"events": events,
	})
}

// AppendEvent validates and appends a new domain event to the stream.
//
//	POST /api/events
func (h *Handler) AppendEvent(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Malformed JSON payload"})
		return
	}
	if req.StreamID == "" {
		req.StreamID = "acc_prime_001"
	}

	// Domain guard — rehydrate current state before accepting command
	currentEvents, _ := h.store.LoadStream(req.StreamID)
	currentState, _ := eventstore.RehydrateAccount(currentEvents)

	if req.EventType == "MONEY_WITHDRAWN" {
		if currentState.Status == "FROZEN" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "Transaction Rejected: Account is FROZEN by compliance",
			})
			return
		}
		if currentState.Balance < req.Amount {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": fmt.Sprintf("Transaction Rejected: Insufficient Funds (Requested $%.2f, Available $%.2f)", req.Amount, currentState.Balance),
			})
			return
		}
	}

	var payload interface{}
	switch req.EventType {
	case "ACCOUNT_OPENED":
		payload = map[string]interface{}{
			"account_id":      req.StreamID,
			"owner":           req.Owner,
			"initial_balance": req.Amount,
		}
	case "MONEY_DEPOSITED", "MONEY_WITHDRAWN":
		if req.Reason == "" {
			req.Reason = "Standard Wire Transfer"
		}
		payload = map[string]interface{}{"amount": req.Amount, "reason": req.Reason}
	case "ACCOUNT_FROZEN", "ACCOUNT_UNFROZEN":
		payload = map[string]interface{}{
			"triggered_by": "Compliance Daemon",
			"timestamp":    time.Now().Unix(),
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unknown event type"})
		return
	}

	evt, err := h.store.Append(req.StreamID, req.EventType, payload, req.ExpectedVersion)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	updatedEvents, _ := h.store.LoadStream(req.StreamID)
	newState, _ := eventstore.RehydrateAccount(updatedEvents)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"new_event":  evt,
		"state":      newState,
		"events_len": len(updatedEvents),
	})
}

// Reset clears the entire event ledger (demo/dev only).
//
//	POST /api/reset
func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	_ = h.store.Clear()
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
