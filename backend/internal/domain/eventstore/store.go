// Package eventstore implements the append-only, thread-safe SQLite/file-based
// event store and the aggregate rehydration (fold) logic for the ledger domain.
package eventstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	ErrConcurrencyConflict = errors.New("optimistic concurrency conflict: stream version mismatch")
	ErrInsufficientFunds   = errors.New("domain validation error: insufficient account balance")
	ErrAccountFrozen       = errors.New("domain validation error: account is frozen")
)

// Event represents an immutable domain event in the append-only ledger.
type Event struct {
	ID        int64           `json:"id"`
	StreamID  string          `json:"stream_id"`
	Version   int64           `json:"version"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// BankAccountState represents the reconstituted projection from folding the event stream.
type BankAccountState struct {
	AccountID   string   `json:"account_id"`
	Owner       string   `json:"owner"`
	Balance     float64  `json:"balance"`
	Status      string   `json:"status"` // ACTIVE, FROZEN, CLOSED
	Version     int64    `json:"version"`
	AuditLog    []string `json:"audit_log"`
	LastUpdated int64    `json:"last_updated"`
}

// FileEventStore implements an append-only, thread-safe persistent JSON event store.
type FileEventStore struct {
	mu       sync.Mutex
	filePath string
}

func NewFileEventStore(filePath string) *FileEventStore {
	return &FileEventStore{filePath: filePath}
}

// Append persists an immutable event to disk with optimistic concurrency control.
func (s *FileEventStore) Append(streamID, eventType string, payload interface{}, expectedVersion int64) (*Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadStreamUnsafe(streamID)
	if err != nil {
		return nil, err
	}

	currentVersion := int64(len(existing))
	if expectedVersion >= 0 && currentVersion != expectedVersion {
		return nil, fmt.Errorf("%w (expected v%d, found v%d)", ErrConcurrencyConflict, expectedVersion, currentVersion)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	evt := Event{
		ID:        time.Now().UnixNano(),
		StreamID:  streamID,
		Version:   currentVersion + 1,
		EventType: eventType,
		Payload:   raw,
		Timestamp: time.Now().Unix(),
	}

	line, err := json.Marshal(evt)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return nil, err
	}

	return &evt, nil
}

// LoadStream reads all events for a given stream ID.
func (s *FileEventStore) LoadStream(streamID string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadStreamUnsafe(streamID)
}

func (s *FileEventStore) loadStreamUnsafe(streamID string) ([]Event, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var events []Event
	for _, l := range splitLines(data) {
		if len(l) == 0 {
			continue
		}
		var evt Event
		if err := json.Unmarshal(l, &evt); err == nil {
			if evt.StreamID == streamID {
				events = append(events, evt)
			}
		}
	}
	return events, nil
}

// Clear truncates the event ledger file.
func (s *FileEventStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.WriteFile(s.filePath, []byte{}, 0644)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// RehydrateAccount folds an immutable sequence of past events into the aggregate's current state.
func RehydrateAccount(events []Event) (*BankAccountState, error) {
	state := &BankAccountState{
		Status:   "UNINITIALIZED",
		AuditLog: make([]string, 0),
	}

	for _, evt := range events {
		state.Version = evt.Version
		state.LastUpdated = evt.Timestamp

		switch evt.EventType {
		case "ACCOUNT_OPENED":
			var p struct {
				AccountID string  `json:"account_id"`
				Owner     string  `json:"owner"`
				Initial   float64 `json:"initial_balance"`
			}
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				return nil, err
			}
			state.AccountID = p.AccountID
			state.Owner = p.Owner
			state.Balance = p.Initial
			state.Status = "ACTIVE"
			state.AuditLog = append(state.AuditLog, fmt.Sprintf("[v%d] Account opened for %s with initial deposit $%.2f", evt.Version, p.Owner, p.Initial))

		case "MONEY_DEPOSITED":
			var p struct {
				Amount float64 `json:"amount"`
				Reason string  `json:"reason"`
			}
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				return nil, err
			}
			state.Balance += p.Amount
			state.AuditLog = append(state.AuditLog, fmt.Sprintf("[v%d] Deposited +$%.2f (%s) -> Balance: $%.2f", evt.Version, p.Amount, p.Reason, state.Balance))

		case "MONEY_WITHDRAWN":
			var p struct {
				Amount float64 `json:"amount"`
				Reason string  `json:"reason"`
			}
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				return nil, err
			}
			if state.Status == "FROZEN" {
				return nil, ErrAccountFrozen
			}
			if state.Balance < p.Amount {
				return nil, fmt.Errorf("%w (requested $%.2f, balance $%.2f)", ErrInsufficientFunds, p.Amount, state.Balance)
			}
			state.Balance -= p.Amount
			state.AuditLog = append(state.AuditLog, fmt.Sprintf("[v%d] Withdrew -$%.2f (%s) -> Balance: $%.2f", evt.Version, p.Amount, p.Reason, state.Balance))

		case "ACCOUNT_FROZEN":
			state.Status = "FROZEN"
			state.AuditLog = append(state.AuditLog, fmt.Sprintf("[v%d] Compliance lock: Account FROZEN", evt.Version))

		case "ACCOUNT_UNFROZEN":
			state.Status = "ACTIVE"
			state.AuditLog = append(state.AuditLog, fmt.Sprintf("[v%d] Compliance unlock: Account RESTORED to ACTIVE", evt.Version))
		}
	}

	return state, nil
}
