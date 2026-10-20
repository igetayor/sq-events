# Ige Single-File Event Sourcing & Snapshot Engine

A zero-dependency Go event-sourcing and entity state-reconstruction engine. Rather than persisting mutable database rows, domain mutations are persisted as immutable, append-only domain events. Current entity states are dynamically derived on startup through fold/reduce rehydration patterns.

## Features
- **Immutable Audit Trail**: Append-only event stream recording sequence, timestamp, and JSON payloads.
- **Pure Go**: Zero external CGO dependencies or external database servers.
- **State Reconstitution**: Pure function `RehydrateAccount(events)` rebuilds bank account balance and statuses reliably.

## Quick Start
```bash
# Run CLI demonstration
go run main.go

# Run unit tests
go test -v ./...
```
