package eventstore

import (
	"os"
	"testing"
)

func TestEventStoreAndRehydration(t *testing.T) {
	testFile := "test_events.db"
	defer os.Remove(testFile)

	store := NewFileEventStore(testFile)
	streamID := "test_stream_1"

	_, err := store.Append(streamID, "ACCOUNT_OPENED", map[string]interface{}{
		"account_id":      streamID,
		"owner":           "Test User",
		"initial_balance": 100.0,
	})
	if err != nil {
		t.Fatalf("Failed to append open event: %v", err)
	}

	_, err = store.Append(streamID, "MONEY_DEPOSITED", map[string]interface{}{
		"amount": 50.0,
	})
	if err != nil {
		t.Fatalf("Failed to append deposit event: %v", err)
	}

	_, err = store.Append(streamID, "MONEY_WITHDRAWN", map[string]interface{}{
		"amount": 30.0,
	})
	if err != nil {
		t.Fatalf("Failed to append withdraw event: %v", err)
	}

	events, err := store.LoadStream(streamID)
	if err != nil {
		t.Fatalf("Failed to load stream: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	state, err := RehydrateAccount(events)
	if err != nil {
		t.Fatalf("Failed to rehydrate: %v", err)
	}

	if state.Balance != 120.0 {
		t.Errorf("Expected balance 120.0, got %.2f", state.Balance)
	}
	if state.Version != 3 {
		t.Errorf("Expected version 3, got %d", state.Version)
	}
}
