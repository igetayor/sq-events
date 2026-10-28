package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	eventstore "ige-sqlite-event-sourcing/eventstore"
)

func runDemo() {
	fmt.Println("============================================================")
	fmt.Println("[*] Ige Event Sourcing & Ledger Rehydration Engine")
	fmt.Println("============================================================")

	tempDb := filepath.Join(os.TempDir(), "ige_demo_events.log")
	defer os.Remove(tempDb)

	store := eventstore.NewFileEventStore(tempDb)
	streamID := "acc_fintech_999"

	fmt.Println("\n[1] Appending Immutable Domain Events to Append-Only Ledger:")

	_, _ = store.Append(streamID, "ACCOUNT_OPENED", map[string]interface{}{
		"account_id":      streamID,
		"owner":           "Omotayo Capital Partners",
		"initial_balance": 50000.00,
	}, 0)
	fmt.Println("   + [Event 1] ACCOUNT_OPENED (Initial Balance: $50,000.00)")

	_, _ = store.Append(streamID, "MONEY_DEPOSITED", map[string]interface{}{
		"amount": 15000.00,
		"reason": "Dividend Inflow",
	}, 1)
	fmt.Println("   + [Event 2] MONEY_DEPOSITED (+$15,000.00)")

	_, _ = store.Append(streamID, "MONEY_WITHDRAWN", map[string]interface{}{
		"amount": 8500.00,
		"reason": "Vendor Payout",
	}, 2)
	fmt.Println("   + [Event 3] MONEY_WITHDRAWN (-$8,500.00)")

	_, _ = store.Append(streamID, "ACCOUNT_FROZEN", map[string]interface{}{
		"reason": "Routine AML Audit Flag",
	}, 3)
	fmt.Println("   + [Event 4] ACCOUNT_FROZEN (Compliance Lock)")

	fmt.Println("\n[2] Loading Stream & Rehydrating Aggregate State from Zero:")
	events, err := store.LoadStream(streamID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	state, err := eventstore.RehydrateAccount(events)
	if err != nil {
		fmt.Printf("Rehydration error: %v\n", err)
		return
	}

	fmt.Printf("   - Account ID:      %s\n", state.AccountID)
	fmt.Printf("   - Owner:           %s\n", state.Owner)
	fmt.Printf("   - Current Balance: $%.2f\n", state.Balance)
	fmt.Printf("   - Account Status:  %s\n", state.Status)
	fmt.Printf("   - Current Version: v%d (Folded %d events)\n", state.Version, len(events))

	fmt.Println("\n[3] Full Audit Trail Projection:")
	for _, entry := range state.AuditLog {
		fmt.Printf("   %s\n", entry)
	}

	fmt.Println("\n[+] Event sourcing pattern demonstrated: zero destructive updates, full time-travel capability.")
	fmt.Println("    To launch the interactive visualizer, run: go run . -serve")
}

func main() {
	serve := flag.Bool("serve", false, "Start interactive web server on port 8094")
	port := flag.String("port", "8094", "Port for web server")
	flag.Parse()

	if *serve {
		startServer(*port)
	} else {
		runDemo()
	}
}
