# central-unit

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![5G](https://img.shields.io/badge/5G-CU--CP-orange.svg)](https://www.3gpp.org/)

## Abstract

This module implements the **gNB Central Unit - Control Plane (CU-CP)** as defined in 3GPP TS 38.401 for 5G NR Open RAN architectures. The CU-CP serves as the control-plane anchor for the gNB, orchestrating radio resource control (RRC) procedures across F1-C, E1, and N2 interfaces.

The implementation follows an event-driven architecture with a central RRC task coordinating protocol handlers for NGAP (AMF interface), F1AP (DU interface), and E1AP (CU-UP interface), mirroring the ITTI message-passing paradigm established in OpenAirInterface (OAI).

## System Architecture

```
                    ┌─────────────────────────────────────────┐
                    │              5G Core Network            │
                    │                   AMF                   │
                    └─────────────────┬───────────────────────┘
                                      │ N2 (NGAP)
                                      │ SCTP PPID=60
                    ┌─────────────────▼───────────────────────┐
                    │                                       │
                    │           CU-CP (this module)         │
                    │         ┌───────────────────┐         │
                    │         │    RRC Task       │         │
                    │         │  (orchestration)  │         │
                    │         └────────┬──────────┘         │
                    │                  │                     │
                    │    ┌─────────────┼─────────────┐      │
                    │    │             │             │      │
                    │ ┌──▼──┐      ┌───▼───┐     ┌───▼───┐  │
                    │ │NGAP │      │ F1AP  │     │ E1AP  │  │
                    │ └─────┘      └───────┘     └───────┘  │
                    └───┬──────────────┬─────────────┬──────┘
                        │              │             │
                        │ F1-C         │ E1          │
                        │ PPID=62      │ PPID=62     │
                    ┌───▼───┐      ┌───▼───┐         │
                    │  DU   │      │ CU-UP │         │
                    └───────┘      └───────┘         │
```

**Key Architectural Components:**

| Component | Responsibility |
|-----------|----------------|
| RRC Task | Central orchestrator for all control-plane procedures |
| NGAP Handler | N2 interface toward AMF (UE context, NAS transport) |
| F1AP Handler | F1-C interface toward DU (RRC transfer, F1 setup) |
| E1AP Handler | E1 interface toward CU-UP (bearer management) |
| Context Manager | UE, DU, and AMF state maintenance |
| Transport Layer | SCTP connection management (PPID filtering) |

## Prerequisites

### Operating System Requirements

This implementation requires **SCTP (Stream Control Transmission Protocol)** kernel support:

| Platform | Support Status |
|----------|---------------|
| Linux (kernel 2.6+) | Full support |
| FreeBSD | Full support |
| macOS | Partial (limitations apply) |
| Windows | Not supported |

### Dependencies

**Ubuntu/Debian:**

```bash
sudo apt-get update
sudo apt-get install -y libsctp-dev lksctp-tools
```

Verify SCTP module availability:

```bash
lsmod | grep sctp
# If not loaded:
sudo modprobe sctp
```

### Runtime Requirements

- **Go**: 1.24.4 or later
- **Network**: IP connectivity to AMF and DU endpoints

## Quick Start

### Build

```bash
git clone <repository-url>
cd central-unit
go mod download
go build -o cucp ./cmd/main.go
```

### Configuration

Create a minimal configuration file (see [Configuration Reference](docs/configuration.md) for complete schema):

```yaml
cucp:
  node_id: "0001"
  node_name: "gNB-CU-CP"
  plmn:
    mcc: "999"
    mnc: "70"
    mnc_length: 2

f1ap:
  local_address: "192.168.1.10"
  local_port: 38472

ngap:
  gnb_id: "000001"
  amf_address: "192.168.1.15"
  amf_port: 38412

logging:
  level: "info"
  format: "json"
```

### Execute

```bash
# Using default configuration path
./cucp -config config/config.yml

# Alternative: direct execution
go run ./cmd/main.go -config config/config.yml
```

### Expected Behavior

Upon successful initialization, the CU-CP will:

1. Load and validate configuration parameters
2. Initialize the logging subsystem
3. Instantiate the CU-CP context (UE/DU/AMF maps)
4. Bind F1AP server on configured endpoint (awaiting DU connections)
5. Establish NGAP association with AMF
6. Enter event-processing loop

Graceful shutdown is triggered via `SIGINT` (Ctrl+C) or `SIGTERM`.

## Project Structure

```
central-unit/
├── cmd/main.go                 # Entry point, signal handling
├── config/config.yml           # Default configuration
├── internal/
│   ├── app/                    # Application lifecycle management
│   ├── common/                 # Shared utilities (FSM, logger, ASN.1)
│   ├── context/                # Protocol orchestration core
│   │   ├── protocol_*.go       # F1AP, NGAP, RRC handlers
│   │   ├── handle_*.go         # AMF, DU, UE event dispatchers
│   │   ├── amfcontext/         # AMF connection state
│   │   ├── du/                 # DU context and F1AP encoding
│   │   └── uecontext/          # UE state machine, security context
│   └── transport/              # SCTP server/client implementation
├── pkg/
│   ├── config/                 # Configuration parsing and validation
│   └── model/                  # Shared type definitions
└── docs/                       # Extended documentation
```

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/architecture.md) | Detailed system architecture, protocol flows, and implementation rationale |
| [Configuration Reference](docs/configuration.md) | Complete configuration schema with parameter descriptions |
| [Development Roadmap](docs/roadmap.md) | Current implementation status and planned features |

## Standards Compliance

This implementation references the following 3GPP specifications:

- **TS 38.401**: NG-RAN Architecture Description
- **TS 38.413**: NG Application Protocol (NGAP)
- **TS 38.473**: F1 Application Protocol (F1AP)
- **TS 38.463**: E1 Application Protocol (E1AP)
- **TS 38.331**: NR Radio Resource Control (RRC)

## License

MIT License. See [LICENSE](LICENSE) for details.
