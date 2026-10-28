# Senior Engineering Walkthrough: Ige SQLite Event Sourcing

> **Target Audience:** Principal Systems Engineers, FinTech Architects, Enterprise Core Banking Leads.  
> **Key Architecture Highlights:** Append-Only Immutable Event Ledger, Deterministic Stream Folding (Rehydration), Optimistic Concurrency Control (`expected_version`), Zero-Destructive Mutation.

---

## 1. Executive Summary & Problem Space

In conventional CRUD architectures, database records are mutated in place (`UPDATE accounts SET balance = 5000 WHERE id = 123`). This results in catastrophic loss of temporal fidelity:
- Why did the balance change?
- Who changed it?
- What was the balance last Tuesday at 2:15 PM?

The **Event Sourcing Pattern** replaces in-place mutations with an immutable, append-only ledger of domain facts (`ACCOUNT_OPENED`, `MONEY_DEPOSITED`, `MONEY_WITHDRAWN`, `ACCOUNT_FROZEN`). Current entity state is a deterministic mathematical projection formed by folding past events in sequence:
$$\text{State}_N = \sum_{i=1}^N f(\text{Event}_i)$$

---

## 2. File & Function Code Architecture

### `eventstore/store.go` — Append-Only Ledger & Rehydration Engine
- `type Event struct`: Immutable event packet containing `ID`, `StreamID`, monotonically incrementing `Version`, `EventType`, and `Payload`.
- `type BankAccountState struct`: In-memory projection containing `AccountID`, `Owner`, calculated `Balance`, `Status` (ACTIVE / FROZEN), and chronological `AuditLog`.
- `type FileEventStore struct`: Thread-safe, disk-backed JSON line event store.
  - `Append(streamID, eventType, payload, expectedVersion)`: Enforces optimistic concurrency control ($O(1)$ lock) by rejecting writes if the stream version has diverged.
  - `LoadStream(streamID)`: Reads all historical events for an aggregate.
  - `Clear()`: Truncates test log.
- `RehydrateAccount(events []Event) (*BankAccountState, error)`: Pure fold function iterating over events from version 1 to $N$, mutating the projection in memory according to domain rules.

### `server.go` — Embedded Zero-Dependency HTTP Gateway (Port 8094)
- `handleGetAccount(w, r)`: Rehydrates and returns the live account projection alongside the full event ledger for visual display.
- `handleAppendEvent(w, r)`: Enforces business validation (e.g. rejecting withdrawals if frozen or if requested amount exceeds current rehydrated balance) before appending the event.
- `handleReset(w, r)`: Truncates ledger.
- `startServer(port)`: Boots HTTP server on port 8094.

### `main.go` — Dual CLI Runner & Server Bootstrap
- `runDemo()`: Terminal execution demonstrating event appending, time-travel fold, and full audit logs.
- `main()`: Parses `-serve` and `-port` flags.

### `static/` — Modular Frontend Visualizer
- `static/css/styles.css`: Dark mode, green/orange/red ledger borders, responsive split view.
- `static/js/api.js`: REST client.
- `static/js/components/AccountCard.js`: Large rehydrated balance header with status badge and aggregate version counter.
- `static/js/components/EventTimeline.js`: Chronological stream showing raw events and formatted financial descriptions.
- `static/js/components/CommandPanel.js`: Interactive actions for Deposit, Withdraw, Freeze, and Unfreeze.
- `static/js/app.js`: Main event orchestrator.

---

## 3. How to Run the Live Demo

### Option A: Interactive Web Mission Control (Port 8094)
```powershell
go run . -serve
# or run the precompiled binary:
.\ige-sqlite-event-sourcing.exe -serve
```
Open your browser to: **http://127.0.0.1:8094**

**Demo Narrative Walkthrough:**
1. Point to the **Rehydrated Projected Balance** card. Notice it displays `$12,500.00` and `Aggregate Version: v2`.
2. Under **Domain Command Dispatcher**, enter `$3,000` and click **+ Deposit Funds**.
   - Show how the event stream instantly gains a new `MONEY_DEPOSITED` item.
   - Show how the balance increases to `$15,500.00` and version advances to `v3`.
3. Try withdrawing `$50,000` (exceeding balance): show that the domain logic rejects the transaction with an explicit error: `Transaction Rejected: Insufficient Funds`.
4. Click **🔒 Freeze Account**:
   - The status badge snaps to **FROZEN**.
   - Try withdrawing `$500`: show that withdrawals are strictly blocked while frozen.
5. Click **🔓 Unfreeze Account**: account returns to **ACTIVE**, and withdrawals succeed.

### Option B: Terminal CLI Execution
```powershell
.\ige-sqlite-event-sourcing.exe
```
Prints the state fold and audit trail to the terminal in pure ASCII.
