package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("============================================================")
	fmt.Println("⚡ Ige Single-File Event Sourcing & State Reconstitution Engine")
	fmt.Println("============================================================")

	dbFile := "events.db"
	_ = os.Remove(dbFile) // Fresh run
	store := NewFileEventStore(dbFile)

	accountID := "acc_99018"

	fmt.Println("\n[1] Appending immutable domain events...")

	// Event 1: Opened
	_, _ = store.Append(accountID, "ACCOUNT_OPENED", map[string]interface{}{
		"account_id":      accountID,
		"owner":           "Adeyinka Ige",
		"initial_balance": 500.0,
	})
	fmt.Println(" - Appended Event: ACCOUNT_OPENED (Initial: $500.00)")

	// Event 2: Deposited
	_, _ = store.Append(accountID, "MONEY_DEPOSITED", map[string]interface{}{
		"amount": 250.0,
	})
	fmt.Println(" - Appended Event: MONEY_DEPOSITED (+$250.00)")

	// Event 3: Withdrawn
	_, _ = store.Append(accountID, "MONEY_WITHDRAWN", map[string]interface{}{
		"amount": 100.0,
	})
	fmt.Println(" - Appended Event: MONEY_WITHDRAWN (-$100.00)")

	fmt.Println("\n[2] Reconstituting Account State from raw event log...")
	events, err := store.LoadStream(accountID)
	if err != nil {
		panic(err)
	}

	state, err := RehydrateAccount(events)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n✨ Reconstituted State:\n")
	fmt.Printf("   Account ID:      %s\n", state.AccountID)
	fmt.Printf("   Account Holder:  %s\n", state.Owner)
	fmt.Printf("   Final Balance:   $%.2f\n", state.Balance)
	fmt.Printf("   Account Status:  %s\n", state.Status)
	fmt.Printf("   Version Applied: %d events\n", state.Version)

	fmt.Println("\nZero external database servers required. Run instantly with 'go run main.go'!")
}
