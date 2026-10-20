package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Event represents an immutable domain event.
type Event struct {
	ID        int64           `json:"id"`
	StreamID  string          `json:"stream_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// BankAccountState represents the reconstituted state from event folding.
type BankAccountState struct {
	AccountID string  `json:"account_id"`
	Owner     string  `json:"owner"`
	Balance   float64 `json:"balance"`
	Status    string  `json:"status"` // ACTIVE, FROZEN, CLOSED
	Version   int64   `json:"version"`
}

// FileEventStore implements an append-only, zero-dependency persistent JSON/log event store.
type FileEventStore struct {
	mu       sync.Mutex
	filePath string
}

func NewFileEventStore(filePath string) *FileEventStore {
	return &FileEventStore{filePath: filePath}
}

// Append persists an immutable event to disk.
func (s *FileEventStore) Append(streamID, eventType string, payload interface{}) (*Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	evt := Event{
		ID:        time.Now().UnixNano(),
		StreamID:  streamID,
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

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var events []Event
	lines := splitLines(data)
	for _, l := range lines {
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

// RehydrateAccount folds past events into current entity state.
func RehydrateAccount(events []Event) (*BankAccountState, error) {
	state := &BankAccountState{
		Status: "UNINITIALIZED",
	}

	for _, evt := range events {
		state.Version++
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

		case "MONEY_DEPOSITED":
			var p struct {
				Amount float64 `json:"amount"`
			}
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				return nil, err
			}
			state.Balance += p.Amount

		case "MONEY_WITHDRAWN":
			var p struct {
				Amount float64 `json:"amount"`
			}
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				return nil, err
			}
			if state.Balance < p.Amount {
				return nil, fmt.Errorf("insufficient funds at event %d", evt.ID)
			}
			state.Balance -= p.Amount

		case "ACCOUNT_FROZEN":
			state.Status = "FROZEN"
		}
	}

	return state, nil
}
