# Architecture

## CU-CP in O-RAN Split Architecture

The Central Unit - Control Plane (CU-CP) constitutes the control-plane entity of a 5G gNB as specified in 3GPP TS 38.401. Within the O-RAN split architecture, the CU-CP occupies the following position:

```
┌─────────────────────────────────────────────────────────────────┐
│                         gNB                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                      CU-CP (this module)                 │    │
│  │            RRC, NGAP, F1AP, E1AP Processing              │    │
│  └─────────────────────────┬───────────────────────────────┘    │
│                            │ E1 (Bearer Management)              │
│  ┌─────────────────────────▼───────────────────────────────┐    │
│  │                         CU-UP                            │    │
│  │              User Plane, SDAP, PDCP                      │    │
│  └─────────────────────────┬───────────────────────────────┘    │
│                            │ F1-U (User Plane)                   │
│  ┌─────────────────────────▼───────────────────────────────┐    │
│  │                          DU                              │    │
│  │               RLC, MAC, PHY, RF                          │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Interface Responsibilities:**

| Interface | Protocol | Remote Entity | Purpose |
|-----------|----------|---------------|---------|
| N2 | NGAP | AMF | UE context management, NAS transport, paging |
| F1-C | F1AP | DU | RRC message transfer, F1 setup, UE context |
| E1 | E1AP | CU-UP | Bearer context setup, QoS flow management |

## Internal Architecture

### Event-Driven Processing Model

The CU-CP implements an event-driven architecture inspired by the OpenAirInterface ITTI (Inter-Task Interface) framework. A central RRC task serves as the orchestration point for all protocol procedures.

```
                    Event Flow
                    ──────────

    ┌──────────┐         ┌──────────┐         ┌──────────┐
    │  NGAP    │         │  F1AP    │         │  E1AP    │
    │ Transport│         │ Transport│         │ Transport│
    └────┬─────┘         └────┬─────┘         └────┬─────┘
         │                    │                    │
         │ SCTP Read          │ SCTP Read          │ SCTP Read
         ▼                    ▼                    ▼
    ┌─────────────────────────────────────────────────────┐
    │              Protocol Handlers                       │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
    │  │ protocol_   │  │ protocol_   │  │ protocol_   │  │
    │  │ ngap.go     │  │ f1c.go      │  │ e1ap.go     │  │
    │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │
    └─────────┼────────────────┼────────────────┼─────────┘
              │                │                │
              │    Events      │                │
              └────────────────┼────────────────┘
                               ▼
                    ┌──────────────────┐
                    │    RRC Task      │
                    │  (Orchestrator)  │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
        ┌──────────┐  ┌──────────┐  ┌──────────┐
        │  UE FSM  │  │  DU FSM  │  │ AMF FSM  │
        │  State   │  │  State   │  │  State   │
        └──────────┘  └──────────┘  └──────────┘
```

### Component Descriptions

#### RRC Task

The RRC task functions as the central coordinator for all control-plane procedures. It:

- Receives decoded protocol messages from transport handlers
- Dispatches events to appropriate FSM instances
- Coordinates multi-step procedures (e.g., UE registration)
- Manages context lifecycle (creation, update, deletion)

#### Protocol Handlers

| Handler | File | Responsibility |
|---------|------|----------------|
| `protocol_ngap.go` | NGAP message processing | NG Setup, Initial UE, DL NAS Transport |
| `protocol_f1c.go` | F1AP message processing | F1 Setup, UL/DL RRC Transfer |
| `protocol_rrc.go` | RRC message construction | RRC Setup, Reconfiguration |
| `protocol_e1ap.go` | E1AP message processing | E1 Setup (planned) |

#### Context Management

The `CuCpContext` struct maintains state for all connected entities:

```go
type CuCpContext struct {
    Ues   map[int64]*uecontext.GNBUe   // RAN UE NGAP ID → UE Context
    Dus   map[int64]*du.GNBDU          // Cell ID → DU Context
    Amfs  map[string]*amfcontext.GNBAmf // Endpoint → AMF Context
}
```

**Note:** Current implementation lacks mutex protection for concurrent access. This is a known limitation requiring resolution for production deployment.

#### Transport Layer

SCTP connections are managed separately for client (NGAP) and server (F1AP/E1AP) roles:

| Component | File | Role | PPID |
|-----------|------|------|------|
| `sctpclient.go` | NGAP client to AMF | Client | 60 |
| `sctpserver.go` | F1AP/E1AP server | Server | 62 |

**Implementation Status:** The F1AP server (`sctpserver.go`) is currently commented out pending full implementation.

## Message Flow Examples

### F1 Setup Procedure

```
DU                              CU-CP                           AMF
 │                                │                              │
 │──── F1SetupRequest ────────────▶                              │
 │    (GNB-DU-ID, Cells)          │                              │
 │                                │                              │
 │◀─── F1SetupResponse ───────────│                              │
 │    (Cells to Activate)         │                              │
 │                                │                              │
 │                                │──── NG Setup Request ────────▶
 │                                │                              │
 │                                │◀─── NG Setup Response ───────│
```

### Initial UE Access

```
DU                              CU-CP                           AMF
 │                                │                              │
 │──── UL RRC Message Transfer ───▶                              │
 │    (RRCSetupRequest)           │                              │
 │                                │                              │
 │                                │──── Initial UE Message ──────▶
 │                                │    (RRCSetupRequest)         │
 │                                │                              │
 │                                │◀─── Downlink NAS Transport ──│
 │                                │    (Registration Accept)     │
 │                                │                              │
 │◀─── DL RRC Message Transfer ───│                              │
 │    (RRCSetup + NAS)            │                              │
```

## Implementation Notes

### OAI ITTI Compatibility

This implementation mirrors the OpenAirInterface ITTI message-passing architecture. Event names and procedures align with OAI conventions to facilitate interoperability and code comprehension for developers familiar with the OAI codebase.

### ASN.1 Encoding

Protocol encoding leverages the `lvdund/ngap` and `lvdund/rrc` libraries for ASN.1 APER encoding/decoding. No external code generation tools are employed; the libraries provide direct Go struct mapping.

### Security Context

The `uecontext` package includes a Milenage algorithm implementation for authentication. This duplicates functionality present in the DU-UE module—a deliberate design choice for module isolation.

### Known Limitations

| Limitation | Location | Status |
|------------|----------|--------|
| No mutex on context maps | `context_cucp.go:142` | TODO |
| F1AP server not implemented | `sctpserver.go` | Commented out |
| RRC Resume/Reestablishment | `protocol_f1c.go:151` | TODO |
| Security context derivation | `handle_amf.go:218,226,227` | TODO |
| E1AP implementation | `protocol_e1ap.go` | Not implemented |

## Threading Model

The current implementation processes events in a single-threaded logic loop. This design choice:

- Simplifies state management
- Eliminates race conditions on context structures
- Provides predictable message ordering

**Future Work:** Introduction of sharded UE context storage with per-shard locking to enable parallel event processing for high-throughput scenarios.
