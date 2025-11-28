# central-unit

Golang control-plane skeleton for a gNB CU-CP. The codebase is structured around an event dispatcher feeding a single logic loop that mirrors the OAI CU-CP flow (RRC core orchestrating NGAP/F1AP/E1AP).

## Layout
- `cmd/cu-cp`: main binary; loads config and wires transports/logic.
- `config`: YAML-backed configuration with validation.
- `internal/app`: lifecycle orchestration for logic + transports.
- `internal/context`: UE/DU/CU-UP/AMF stores and mapping indexes.
- `internal/logic`: event handlers and RRC state transitions.
- `internal/transport`: dispatcher plus NGAP/F1AP/E1AP transport stubs.
- `internal/obs`: logging/metrics/tracing placeholders.
- `internal/statemachine`: RRC state machine.
- `internal/timers`: timer wheel placeholder.

## Running
```bash
go run ./cmd/cu-cp -config config/config.yml
```
The binary currently wires validated config, logging, and stubbed transports; protocol handling is scaffolded for future implementation.
