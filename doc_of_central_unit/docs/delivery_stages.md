# Central Unit Delivery Stages

## Stage 0 — Scaffolding & Tooling

- **Deliverables:** Repository skeleton, `go.mod`, lint/test wiring, placeholder packages for context, protocols, services, and transport.
- **Readiness Criteria:** CI compiles empty shell; docs (`structure.md`) outline the target module map.

## Stage 1 — Configuration & Bootstrap

- **Deliverables:** Config parser (`config/config.go`), validation, logger/metric initialization, and a `cmd/main.go` that wires the bootstrap sequence.
- **Readiness Criteria:** Binary loads `config.yml`, exits gracefully with structured logs, and exposes health endpoints (HTTP stub acceptable).

## Stage 2 — Transport Layer

- **Deliverables:** SCTP abstraction in `internal/transport/sctp`, HTTP control-plane surface, connection lifecycle management, and retry/backoff policies.
- **Readiness Criteria:** CU listens on F1/E1/NG bind addresses from config, emits connection metrics, and supports dependency injection for protocol handlers.

## Stage 3 — Protocol Tasks

- **Deliverables:** Task loops for NGAP, F1AP, and E1AP plus dispatcher scaffolds; message encoders/decoders leveraging ASN.1 toolchain.
- **Readiness Criteria:** CU can establish NGAP association with an AMF and complete F1 Setup with at least one DU in a lab environment.

## Stage 4 — RRC & UE Lifecycle

- **Deliverables:** UE state machines, bearer/context coordination, RRC procedure coverage per `docs/note_interface_rrc.md`, and hooks into PDCP/SDAP as needed.
- **Readiness Criteria:** End-to-end attach (RRC Setup → NG attach) in simulation, supporting resume/reestablishment and paging flows.

## Stage 5 — Operations & Hardening

- **Deliverables:** Observability (metrics, tracing, logging enrichment), chaos/testing harnesses, configuration hot-reload, and deployment manifests.
- **Readiness Criteria:** System sustains long-running sessions with automated rollouts, alerts, and documented SLOs.
