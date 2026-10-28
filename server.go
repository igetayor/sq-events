package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	eventstore "ige-sqlite-event-sourcing/eventstore"
)

const defaultPort = "8094"

var store *eventstore.FileEventStore

type CommandRequest struct {
	StreamID        string  `json:"stream_id"`
	EventType       string  `json:"event_type"` // ACCOUNT_OPENED, MONEY_DEPOSITED, MONEY_WITHDRAWN, ACCOUNT_FROZEN, ACCOUNT_UNFROZEN
	Owner           string  `json:"owner"`
	Amount          float64 `json:"amount"`
	Reason          string  `json:"reason"`
	ExpectedVersion int64   `json:"expected_version"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleGetAccount(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	streamID := r.URL.Query().Get("stream_id")
	if streamID == "" {
		streamID = "acc_prime_001"
	}

	events, err := store.LoadStream(streamID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// If no events exist yet, seed initial account
	if len(events) == 0 {
		_, _ = store.Append(streamID, "ACCOUNT_OPENED", map[string]interface{}{
			"account_id":      streamID,
			"owner":           "Ige Institutional Vault",
			"initial_balance": 10000.00,
		}, -1)
		_, _ = store.Append(streamID, "MONEY_DEPOSITED", map[string]interface{}{
			"amount": 2500.00,
			"reason": "Initial Treasury Allocation",
		}, -1)
		events, _ = store.LoadStream(streamID)
	}

	state, err := eventstore.RehydrateAccount(events)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]interface{}{
		"state":  state,
		"events": events,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleAppendEvent(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Malformed JSON payload"})
		return
	}

	if req.StreamID == "" {
		req.StreamID = "acc_prime_001"
	}

	// Domain validation: rehydrate current state first
	currentEvents, _ := store.LoadStream(req.StreamID)
	currentState, _ := eventstore.RehydrateAccount(currentEvents)

	if req.EventType == "MONEY_WITHDRAWN" {
		if currentState.Status == "FROZEN" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"error": "Transaction Rejected: Account is FROZEN by compliance"})
			return
		}
		if currentState.Balance < req.Amount {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{
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
		payload = map[string]interface{}{
			"amount": req.Amount,
			"reason": req.Reason,
		}
	case "ACCOUNT_FROZEN", "ACCOUNT_UNFROZEN":
		payload = map[string]interface{}{
			"triggered_by": "Compliance Daemon",
			"timestamp":    time.Now().Unix(),
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown event type"})
		return
	}

	evt, err := store.Append(req.StreamID, req.EventType, payload, req.ExpectedVersion)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Re-fold stream to return new projected state
	updatedEvents, _ := store.LoadStream(req.StreamID)
	newState, _ := eventstore.RehydrateAccount(updatedEvents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"new_event":  evt,
		"state":      newState,
		"events_len": len(updatedEvents),
	})
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	_ = store.Clear()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func startServer(port string) {
	workDir, _ := os.Getwd()
	dbPath := filepath.Join(workDir, "events.db")
	store = eventstore.NewFileEventStore(dbPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/account", handleGetAccount)
	mux.HandleFunc("/api/events", handleAppendEvent)
	mux.HandleFunc("/api/reset", handleReset)

	staticDir := filepath.Join(workDir, "static")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = workDir
	}

	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	addr := ":" + port
	fmt.Printf("[*] SQLite Event Sourcing Server running at http://127.0.0.1%s\n", addr)
	fmt.Println("    Endpoints: GET /api/account | POST /api/events | POST /api/reset")

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server error: %v", err)
	}
}
