<div align="center">

# 🚀 central-unit

**A Golang implementation of a gNB Central Unit - Control Plane (CU-CP) for 5G Open RAN architecture**

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![5G](https://img.shields.io/badge/5G-CU--CP-orange.svg)](https://www.3gpp.org/)

---

*A Golang implementation of a gNB Central Unit - Control Plane (CU-CP) for 5G Open RAN architecture. This project implements an RRC core that orchestrates NGAP (AMF interface), F1AP (DU interface), and E1AP (CU-UP interface) protocols, mirroring the OAI (OpenAirInterface) CU-CP logic flow.*

</div>

CU-CP is organized as:

- **Central RRC task**: Orchestrates procedures
- **Protocol tasks**: Handle SCTP and encode/decode
- **Context storage**: Maintains UE/DU/CU-UP state
- **Event-driven**: ITTI messages coordinate between tasks

The RRC task is the core, processing events from F1AP (DU), NGAP (AMF), and E1AP (CU-UP) to manage UE connections and radio resources.

## Overview

The CU-CP is the control plane component of a 5G gNB that manages:
- **UE Context**: RRC state machine, security context, bearer management
- **DU Management**: F1AP interface for Distributed Unit connections
- **AMF Communication**: NGAP interface for Access and Mobility Management Function
- **Protocol Handling**: RRC, NGAP, F1AP message processing and routing

The codebase follows an event-driven architecture with a single-threaded logic loop processing events from transports, targeting scalability for high-throughput control-plane operations.

## Requirements

### Operating System Support

This project requires **SCTP (Stream Control Transmission Protocol)** support, which is available on:
- Linux (kernel 2.6+)
- FreeBSD
- macOS (with limitations)

**Note**: SCTP is not natively supported on Windows.

### Ubuntu/Debian Package Installation

Install required SCTP development libraries:

```bash
sudo apt-get update
sudo apt-get install -y libsctp-dev lksctp-tools
```

Verify SCTP support:
```bash
# Check if SCTP module is loaded
lsmod | grep sctp

# If not loaded, load it manually
sudo modprobe sctp
```

### Go Version

Requires Go 1.24.4 or later.

## Configuration

The CU-CP is configured via a YAML file (`config/config.yml`). Here's the configuration structure:

### Configuration File Structure

```yaml
cucp:
  node_id: "0001"              # CU-CP node identifier
  node_name: "gNB-CU-CP"        # CU-CP node name
  plmn:
    mcc: "999"                  # Mobile Country Code
    mnc: "70"                   # Mobile Network Code
    mnc_length: 2               # MNC length (2 or 3 digits)
  slices:                       # Network Slice Support
    - sst: "01"                 # Slice/Service Type
      sd: "010203"              # Slice Differentiator
  tac: "000001"                 # Tracking Area Code

f1ap:                           # F1AP interface (CU-CP <-> DU)
  local_address: "192.168.1.10" # Local IP address for F1AP server
  local_port: 38472             # Local port for F1AP server
  sctp:
    in_streams: 2               # SCTP inbound streams
    out_streams: 2              # SCTP outbound streams
  timers:
    f1_setup_timer: "10s"       # F1 Setup timer duration

e1ap:                           # E1AP interface (CU-CP <-> CU-UP)
  local_address: "192.168.1.10" # Local IP address for E1AP server
  local_port: 38462             # Local port for E1AP server
  sctp:
    in_streams: 2               # SCTP inbound streams
    out_streams: 2              # SCTP outbound streams

ngap:                           # NGAP interface (CU-CP <-> AMF)
  gnb_id: "000001"              # gNB identifier
  amf_address: "192.168.1.15"   # AMF IP address
  amf_port: 38412               # AMF port
  local_address: "192.168.1.10"  # Local IP address for NGAP client
  local_port: 9487              # Local port for NGAP client
  sctp:
    in_streams: 2               # SCTP inbound streams
    out_streams: 2              # SCTP outbound streams

logging:
  level: "info"                 # Log level: debug, info, warn, error
  format: "json"                # Log format: json or text
```

### Configuration Parameters

#### CU-CP Section
- `node_id`: Unique identifier for this CU-CP instance
- `node_name`: Human-readable name for the CU-CP
- `plmn`: Public Land Mobile Network configuration
  - `mcc`: Mobile Country Code (3 digits)
  - `mnc`: Mobile Network Code (2 or 3 digits)
  - `mnc_length`: Length of MNC (2 or 3)
- `slices`: List of supported network slices (S-NSSAI)
  - `sst`: Slice/Service Type (hex string)
  - `sd`: Slice Differentiator (hex string, optional)
- `tac`: Tracking Area Code (hex string)

#### F1AP Section
- `local_address`: IP address where CU-CP listens for DU connections
- `local_port`: Port number for F1AP server
- `sctp`: SCTP stream configuration
- `timers`: F1AP-specific timer configurations

#### NGAP Section
- `gnb_id`: gNB identifier (hex string)
- `amf_address`: AMF server IP address
- `amf_port`: AMF server port
- `local_address`: Local IP address for NGAP client connection
- `local_port`: Local port for NGAP client

#### Logging Section
- `level`: Logging verbosity level
- `format`: Output format (json recommended for production)

## Running

### Build and Run

```bash
# Clone the repository
git clone <repository-url>
cd central-unit

# Install dependencies
go mod download

# Run the application
go run ./cmd/main.go -config config/config.yml
```

### Command Line Options

- `-config`: Path to configuration file (default: `config/config.yml`)

### Example Run

```bash
# Using default config path
go run ./cmd/main.go

# Using custom config path
go run ./cmd/main.go -config /path/to/config.yml
```

The application will:
1. Load and validate the configuration
2. Initialize logging
3. Create CU-CP context
4. Start F1AP server (listening for DU connections)
5. Connect to AMF via NGAP
6. Process incoming messages

Press `Ctrl+C` to gracefully shutdown the application.

## Features & Future Works

### Planned Features

The following features are planned for future releases:

1. **E1AP Message Handler**
   - E1 Setup Request/Response procedures
   - Bearer Context Setup/Modification/Release
   - CU-UP connection management
   - E1AP message encoding/decoding

2. **Multi-Connection Support**
   - Multiple DU connections management
   - Multiple UE context handling
   - Multiple AMF associations support
   - Connection pooling and load balancing

3. **Extended Procedures**
   - **Session Management**: PDU Session Establishment/Modification/Release
   - **Handover**: Intra-DU, Inter-DU, and Inter-gNB handover procedures
   - **Paging**: Paging request handling and UE paging
   - **UE Context Release**: UE context release procedures
   - **RRC Reestablishment**: RRC connection reestablishment
   - **RRC Resume**: RRC connection resume procedures

4. **Security Context Support**
   - Complete 5G-AKA authentication flow
   - Security key derivation (KgNB, KUPenc, KUPint)
   - Integrity protection and verification
   - Ciphering/deciphering of RRC and NAS messages
   - Security mode command handling

