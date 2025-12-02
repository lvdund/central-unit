# CU-CP Project Planning and Architecture

## Project Description

**central-unit** is a Golang implementation of a gNB Central Unit - Control Plane (CU-CP) for 5G Open RAN architecture. The project mirrors the OAI (OpenAirInterface) CU-CP logic, implementing an RRC core that orchestrates NGAP (AMF interface), F1AP (DU interface), and E1AP (CU-UP interface) protocols.

### Key Characteristics
- **Event-driven architecture**: Single-threaded logic loop processing events from transports
- **Scalability target**: 10M+ control-plane operations per second
- **Protocol libraries**: Uses existing Go libraries for NGAP (`github.com/lvdund/ngap`), F1AP (`github.com/JocelynWS/f1-gen`), and RRC (`github.com/lvdund/rrc`)
- **SCTP transport**: Client/server implementations for NGAP (AMF), F1AP (DU), and E1AP (CU-UP)
- **State machine**: RRC state machine for UE lifecycle management (Idle ↔ Connected ↔ Connected-Inactive)

---

## Components of CU-CP

### 1. Application Layer (`internal/app`)

**Purpose**: Lifecycle orchestration and configuration management

**Components**:
- **`app.go`**: Main application structure
  - `New(cfgPath)`: Loads and validates configuration from YAML file
  - `Start()`: Initializes CU-CP context, sets up transports, and starts services
  - `Stop(ctx)`: Gracefully shuts down all services

**Responsibilities**:
- Configuration loading and validation
- Logger initialization
- Context creation and initialization
- Service lifecycle management

---

### 2. Context Management (`internal/context`)

**Purpose**: Stores and manages UE, DU, CU-UP, and AMF contexts with secondary indexes

#### 2.1 CU-CP Context (`cucp.go`)

**Structure**: `CuCpContext`
- **Control Information**: PLMN (MCC/MNC), gNB ID/IP/Port, TAC, slice information
- **Context Pools**: 
  - `UePool`: UE contexts indexed by RAN UE NGAP ID
  - `DuPool`: DU contexts indexed by DU ID
  - `AmfPool`: AMF contexts indexed by AMF ID
  - `DuConnPool`: Temporary DU connections indexed by SCTP association ID
- **ID Generators**: UE ID, AMF ID, DU ID, TEID generators
- **Helper Methods**: PLMN conversion, gNB ID conversion, slice conversion, TAC conversion

**Key Functions**:
- `SetControlInfoFromConfig()`: Sets control plane information from config
- `SetSliceInfoFromConfig()`: Sets slice information from config
- `Terminate()`: Cleanup and shutdown

#### 2.2 UE Context (`uecontext/uecontext.go`)

**Structure**: `GNBUe`
- **Identifiers**: 
  - `RanUeNgapId`: UE identifier in gNB context
  - `AmfUeNgapId`: UE identifier in AMF context
  - `PrUeId`: Unique UE ID
- **State**: UE state (INITIALIZED, ONGOING, READY, DOWN)
- **Security**: TMSI, security capabilities
- **Connection**: SCTP connection reference
- **NAS**: NAS message cache

**Purpose**: Maintains per-UE state, security context, and session information

#### 2.3 DU Context (`du/ducontext.go`)

**Structure**: `GNBDU`
- **Identifiers**: `DuId`, `DuName`, `AssocId` (SCTP association ID)
- **State**: DU state (INACTIVE, ACTIVE, LOST)
- **Transport**: SCTP connection (`Tnla.SctpConn`)
- **System Information**: 
  - `MIB`: Master Information Block
  - `SIB1`: System Information Block Type 1
  - `MTC`: Measurement Timing Configuration
- **Cells**: `ServedCells` list with cell information (CellID, PCI, PLMN, TAC)
- **Setup Request**: Deep copy of F1 Setup Request

**Key Methods**:
- `SendF1ap()`: Sends F1AP messages to DU via SCTP
- `GetCellByID()`: Retrieves cell information by cell ID
- `IsActive()`: Checks if DU is in active state

**Purpose**: Manages DU associations, served cells, and system information cache

#### 2.4 AMF Context (`amfcontext/amfcontext.go`)

**Structure**: `GNBAmf`
- **Identifiers**: `AmfId`, `AmfIp`, `AmfPort`
- **State**: AMF state (INACTIVE, ACTIVE, OVERLOADED)
- **Transport**: SCTP connection (`Tnla.SctpConn`)
- **PLMN Support**: List of supported PLMNs
- **Slice Support**: List of supported slices (SST/SD)
- **Capacity**: Relative AMF capacity
- **GUAMI**: AMF region ID, set ID, pointer

**Key Methods**:
- `SendNgap()`: Sends NGAP messages to AMF via SCTP
- `GetPlmnSupport()`: Retrieves PLMN support by index
- `GetSliceSupport()`: Retrieves slice support by index

**Purpose**: Manages AMF associations, PLMN/slice support, and NGAP communication

#### 2.5 Context Initialization (`init.go`)

**Functions**:
- `InitContext()`: Initializes CU-CP context
  - Creates AMF connection (SCTP client)
  - Sends NG Setup Request
  - Initializes F1AP SCTP server for DU connections
- `initAmfConn()`: Establishes SCTP connection to AMF
- `initF1APServer()`: Starts SCTP server listening for DU connections
- `newAmf()`: Creates new AMF context
- `newDu()`: Creates new DU context

---

### 3. Transport Layer (`internal/transport`)

**Purpose**: SCTP-based transport for NGAP, F1AP, and E1AP protocols

#### 3.1 SCTP Client (`sctpclient.go`)

**Structure**: `SctpConn`
- **Connection Management**: Local/remote addresses, SCTP connection handle
- **Read/Write Workers**: Parallel workers for message processing
- **Channels**: `ReadCh` for incoming messages
- **Thread Safety**: Mutex protection for SCTP operations

**Key Methods**:
- `Connect()`: Establishes SCTP connection (for AMF)
- `Read()`: Returns read channel for incoming messages
- `Send()`: Sends data via SCTP
- `Close()`: Closes connection gracefully

**Usage**: NGAP communication with AMF (outgoing connections)

#### 3.2 SCTP Server (`sctpserver.go`)

**Structure**: `F1APServer`
- **Listener**: SCTP listener for incoming connections
- **Connection Management**: `sync.Map` for active connections
- **Configuration**: SCTP init message, read buffer size, timeouts
- **Callback**: `onNewConn` callback for new DU connections

**Key Methods**:
- `Run()`: Starts server and begins accepting connections
- `Stop()`: Gracefully stops server and closes all connections
- `handleConnection()`: Manages individual connection lifecycle
- `Send()`: Sends data to specific connection by association ID

**Usage**: F1AP communication with DUs (incoming connections)

#### 3.3 Message Dispatching

**NGAP Dispatch** (`amfdispatch.go`):
- Decodes NGAP messages using `ngap.NgapDecode`
- Routes messages based on procedure code and present type
- Handles NG Setup Response, Initial UE Message, etc.

**F1AP Dispatch** (`f1dispatch.go`):
- Decodes F1AP messages using `f1ap.F1apDecode`
- Routes messages based on procedure code
- Handles F1 Setup Request, UE Context messages, RRC message transfers

---

### 4. Protocol Handlers (`internal/context/du`)

**Purpose**: F1AP message encoding/decoding and handler functions

#### 4.1 F1AP Decode/Encode (`handlers.go`)

**Functions**:
- `F1apDecode()`: Decodes F1AP PDU bytes using `f1ap.F1apDecode`
  - Returns wrapper `F1apPDU` with Present type and Message
  - Extracts procedure code and message payload
- `SendF1SetupResponse()`: Encodes and sends F1 Setup Response
  - Creates `ies.F1SetupResponse` message
  - Encodes using `msg.Encode()`
  - Sends via SCTP
- `SendF1SetupFailure()`: Encodes and sends F1 Setup Failure
  - Creates `ies.F1SetupFailure` with cause
  - Encodes and sends via SCTP

#### 4.2 F1 Setup Handler (`f1setup.go`)

**Functions**:
- `HandleF1SetupRequest()`: Processes F1 Setup Request from DU
- `ValidateF1SetupRequest()`: Validates setup request according to OAI rules

**Handler Logic** (`f1dispatch.go:handleF1SetupRequest`):
1. Extract transaction ID, DU ID, DU name
2. Validate exactly one cell
3. Validate PLMN matches CU config
4. Check for duplicate DU IDs
5. Check for CellID/PCI clashes
6. Extract MIB, SIB1, MTC
7. Store DU context
8. Send F1 Setup Response with cells to activate

---

### 5. State Machine (`internal/common/fsm`)

**Purpose**: Generic finite state machine library

**Components**:
- **`types.go`**: `StateType` and `EventType` type definitions
- **`state.go`**: State management with current state and event tracking
- **`event.go`**: Event data structure and methods
- **`fsm.go`**: Core FSM logic with transitions and callbacks

**RRC State Machine** (`internal/statemachine/rrc.go`):
- **States**: Idle, Connected, ConnectedInactive, Releasing
- **Events**: PowerOn, AttachRRCConnect, DetachRRCRelease, ConnectionFailure, InactivityTimerExpiry, ResumeRequest, ReestablishmentRequest, ReestablishmentComplete
- **Transitions**: Defined state transitions based on events
- **Callbacks**: State entry/exit handlers

---

### 6. Logging (`internal/common/logger`)

**Purpose**: Structured logging using zerolog

**Features**:
- Console writer with color-coded log levels
- Field-based logging with context
- Log level parsing and configuration
- Methods: Info, Warn, Error, Fatal, Debug, Trace

---

### 7. Configuration (`pkg/config`)

**Purpose**: YAML-based configuration with validation

**Structure**:
- **CUCPConfig**: Node ID, name, PLMN, slices
- **F1APConfig**: Local address/port, SCTP streams, timers
- **E1APConfig**: Local address/port, SCTP streams
- **NGAPConfig**: AMF address/port, local address/port, SCTP streams
- **LoggingConfig**: Level, format

**Functions**:
- `Load()`: Reads and parses YAML config
- `Validate()`: Validates required fields and constraints
- `applyDefaults()`: Applies default values

---

## Contexts Detailed

### UE Context Structure

```go
type GNBUe struct {
    RanUeNgapId    int64    // RAN UE NGAP ID (primary key)
    AmfUeNgapId    int64    // AMF UE NGAP ID
    PrUeId         int64    // Unique UE identifier
    State          uint8    // UE state (INITIALIZED, ONGOING, READY, DOWN)
    Msin           string   // Mobile subscriber identifier
    SctpConnection *SctpConn // SCTP connection reference
    Tmsi           *nas.Guti // Temporary mobile subscriber identity
    // Security context, PDU sessions, DRBs (to be added)
}
```

**Storage**: Sharded maps with secondary indexes:
- Primary: `UePool` (RAN UE NGAP ID)
- Secondary: `MsinPool` (MSIN), `PrUePool` (PrUeId), `TeidPool` (TEID)

### DU Context Structure

```go
type GNBDU struct {
    DuId        int64              // DU ID (GNB-DU-ID)
    DuName      string             // DU name
    AssocId     string             // SCTP association ID
    State       string             // DU state (INACTIVE, ACTIVE, LOST)
    Tnla        TNLAssociation     // SCTP connection
    SetupReq    *ies.F1SetupRequest // F1 Setup Request
    MIB         []byte             // Master Information Block
    SIB1        []byte             // System Information Block 1
    MTC         []byte             // Measurement Timing Configuration
    ServedCells []ServedCell       // List of served cells
}
```

**Storage**: `DuPool` indexed by DU ID, `DuConnPool` indexed by AssocId (temporary)

### AMF Context Structure

```go
type GNBAmf struct {
    AmfId               int64          // AMF identifier
    AmfIp               string         // AMF IP address
    AmfPort             int            // AMF port
    Name                string         // AMF name
    State               string         // AMF state (INACTIVE, ACTIVE, OVERLOADED)
    Tnla                TNLAssociation // SCTP connection
    RelativeAmfCapacity int64          // AMF capacity
    Plmns               *PlmnSupported // Supported PLMNs
    Slices              *SliceSupported // Supported slices
    RegionId            aper.BitString // AMF region ID
    SetId               aper.BitString // AMF set ID
    Pointer             aper.BitString // AMF pointer
}
```

**Storage**: `AmfPool` indexed by AMF ID

---

## Logic Handlers

### NGAP Handlers (`internal/context/amfdispatch.go`)

**Message Flow**:
1. **NG Setup Request** → `SendNgSetupRequest()`
   - Constructs NG Setup Request with PLMN, gNB ID, slices, TAC
   - Encodes using `ngap.NgapEncode`
   - Sends via SCTP to AMF

2. **NG Setup Response** → `handlerNgSetupResponse()`
   - Extracts AMF name, capacity, PLMN support, slice support
   - Updates AMF context
   - Sets AMF state to ACTIVE

3. **Initial UE Message** (to be implemented)
   - Forwards NAS message from UE to AMF
   - Includes RAN UE NGAP ID, AMF UE NGAP ID

4. **Initial Context Setup Request** (to be implemented)
   - Receives security context, UE capabilities from AMF
   - Triggers security mode command
   - Sets up UE context

### F1AP Handlers (`internal/context/f1dispatch.go`)

**Message Flow**:
1. **F1 Setup Request** → `handleF1SetupRequest()`
   - Validates single cell, PLMN match, no duplicates
   - Extracts MIB/SIB1/MTC
   - Stores DU context
   - Sends F1 Setup Response

2. **Initial UL RRC Message Transfer** (to be implemented)
   - Receives RRC Setup Request from DU
   - Creates UE context
   - Generates RRC Setup
   - Sends via F1AP DL RRC Message Transfer

3. **UL RRC Message Transfer** (to be implemented)
   - Receives RRC messages from UE via DU
   - Processes RRC Setup Complete, Security Mode Complete, etc.
   - Triggers state transitions

4. **UE Context Setup Request** (to be implemented)
   - Configures DRBs, security, cell group
   - Sends to DU for UE context establishment

5. **UE Context Release** (to be implemented)
   - Releases UE context on DU
   - Cleans up resources

### RRC Procedures (to be implemented in `internal/logic`)

**Procedures**:
1. **RRC Setup**: Initial connection establishment
2. **RRC Reconfiguration**: DRB setup, security activation
3. **RRC Release**: Connection release
4. **Security Mode Command**: AS security activation
5. **UE Capability Enquiry**: UE capability retrieval
6. **RRC Reestablishment**: Connection recovery

---

## NAS/NGAP/F1/RRC Procedure Signaling Map

### 1. Initial Access Flow

```
UE                    DU                    CU-CP                AMF
 |                     |                      |                   |
 |--[RRC SetupReq]---->|                      |                   |
 |                     |--[F1: Initial UL RRC]-->                  |
 |                     |   Transfer            |                   |
 |                     |                      |--[NG: Initial UE]-->|
 |                     |                      |   Message          |
 |<--[RRC Setup]-------|                      |                   |
 |                     |<--[F1: DL RRC]--------|                   |
 |                     |   Transfer            |                   |
 |--[RRC SetupComplete]|                      |                   |
 |                     |--[F1: UL RRC]-------->|                   |
 |                     |   Transfer            |                   |
 |                     |                      |<--[NG: Initial]---|
 |                     |                      |   Context Setup   |
 |                     |                      |   Request         |
 |<--[SecurityModeCmd]--|                      |                   |
 |                     |<--[F1: DL RRC]--------|                   |
 |                     |   Transfer            |                   |
 |--[SecurityModeComplete]|                    |                   |
 |                     |--[F1: UL RRC]-------->|                   |
 |                     |   Transfer            |                   |
 |                     |                      |--[NG: Initial]-->|
 |                     |                      |   Context Setup  |
 |                     |                      |   Response       |
```

**Key Messages**:
- **F1AP**: Initial UL RRC Message Transfer, DL RRC Message Transfer, UL RRC Message Transfer
- **NGAP**: Initial UE Message, Initial Context Setup Request/Response
- **RRC**: RRCSetupRequest, RRCSetup, RRCSetupComplete, SecurityModeCommand, SecurityModeComplete

### 2. F1 Setup Procedure

```
DU                    CU-CP
 |                      |
 |--[F1 Setup Request]-->|
 |  - DU ID             |
 |  - DU Name           |
 |  - Served Cells      |
 |  - MIB/SIB1/MTC      |
 |  - RRC Version       |
 |                      |
 |<--[F1 Setup Response]|
 |  - Transaction ID    |
 |  - CU Name           |
 |  - Cells to Activate |
 |  - RRC Version       |
```

**Handler**: `handleF1SetupRequest()` in `f1dispatch.go`
**Validation**: Single cell, PLMN match, no duplicate DU ID, no cell/PCI clash

### 3. NG Setup Procedure

```
CU-CP                AMF
 |                    |
 |--[NG Setup Request]|
 |  - Global RAN Node |
 |    ID (PLMN+gNB ID)|
 |  - RAN Node Name   |
 |  - Supported TA    |
 |  - Default Paging  |
 |    DRX             |
 |                    |
 |<--[NG Setup Response]
 |  - AMF Name        |
 |  - PLMN Support    |
 |  - Slice Support   |
 |  - Relative Capacity|
```

**Handler**: `SendNgSetupRequest()` → `handlerNgSetupResponse()`

### 4. UE Context Setup Flow

```
CU-CP                CU-UP                DU                    UE
 |                    |                    |                     |
 |--[E1: Bearer Ctx]-->|                    |                     |
 |   Setup Request    |                    |                     |
 |                    |<--[E1: Bearer Ctx]--|                     |
 |                    |   Setup Response   |                     |
 |                    |                    |                     |
 |                    |                    |--[F1: UE Context]--|
 |                    |                    |   Setup Request     |
 |                    |                    |                     |
 |                    |                    |<--[F1: UE Context]--|
 |                    |                    |   Setup Response    |
 |                    |                    |                     |
 |                    |                    |--[F1: DL RRC]------>|
 |                    |                    |   Transfer          |
 |                    |                    |   (RRCReconfig)     |
 |                    |                    |                     |
 |                    |                    |<--[F1: UL RRC]------|
 |                    |                    |   Transfer          |
 |                    |                    |   (RRCReconfigComplete)
```

**Key Messages**:
- **E1AP**: Bearer Context Setup Request/Response
- **F1AP**: UE Context Setup Request/Response, DL/UL RRC Message Transfer
- **RRC**: RRCReconfiguration, RRCReconfigurationComplete

### 5. PDU Session Setup Flow

```
UE                    CU-CP                AMF                CU-UP
 |                     |                    |                   |
 |--[NAS: PDU Session]|                    |                   |
 |   Establishment    |                    |                   |
 |   Request          |                    |                   |
 |                     |--[NG: UL NAS]------>|                   |
 |                     |   Transport        |                   |
 |                     |                    |--[Nsmf_PDUSession]|
 |                     |                    |   CreateSMContext  |
 |                     |                    |<--[Nsmf_PDUSession]|
 |                     |                    |   CreateSMContext  |
 |                     |                    |   Response        |
 |                     |<--[NG: DL NAS]------|                   |
 |                     |   Transport        |                   |
 |                     |                    |                   |
 |                     |--[NG: PDU Session]--|                   |
 |                     |   Resource Setup   |                   |
 |                     |   Request          |                   |
 |                     |                    |                   |
 |                     |                    |--[E1: Bearer Ctx]-->|
 |                     |                    |   Setup Request   |
 |                     |                    |                   |
 |                     |                    |<--[E1: Bearer Ctx]--|
 |                     |                    |   Setup Response  |
 |                     |                    |                   |
 |<--[RRC Reconfig]----|                    |                   |
 |   (DRB Setup)       |                    |                   |
 |                     |                    |                   |
 |--[RRC ReconfigComplete]|                  |                   |
 |                     |                    |                   |
 |                     |--[NG: PDU Session]--|                   |
 |                     |   Resource Setup   |                   |
 |                     |   Response        |                   |
```

**Key Messages**:
- **NAS**: PDU Session Establishment Request/Response
- **NGAP**: UL/DL NAS Transport, PDU Session Resource Setup Request/Response
- **E1AP**: Bearer Context Setup Request/Response
- **RRC**: RRCReconfiguration, RRCReconfigurationComplete

### 6. UE Context Release Flow

```
AMF                CU-CP                DU                    UE
 |                  |                    |                     |
 |--[NG: UE Context]|                    |                     |
 |   Release Command|                    |                     |
 |                  |                    |                     |
 |                  |--[F1: UE Context]--|                     |
 |                  |   Release Command  |                     |
 |                  |                    |                     |
 |                  |                    |--[RRC Release]----->|
 |                  |                    |                     |
 |                  |                    |<--[RRC Release]-----|
 |                  |                    |   Complete         |
 |                  |                    |                     |
 |                  |<--[F1: UE Context]|                     |
 |                  |   Release Complete|                     |
 |                  |                    |                     |
 |<--[NG: UE Context]|                    |                     |
 |   Release Complete|                    |                     |
```

**Key Messages**:
- **NGAP**: UE Context Release Command/Complete
- **F1AP**: UE Context Release Command/Complete
- **RRC**: RRCRelease, RRCReleaseComplete

### 7. RRC State Transitions

```
[Idle] --PowerOn--> [Idle]
[Idle] --AttachRRCConnect--> [Connected]
[Connected] --DetachRRCRelease--> [Releasing]
[Connected] --ConnectionFailure--> [Idle]
[Connected] --InactivityTimerExpiry--> [ConnectedInactive]
[Connected] --ReestablishmentRequest--> [Connected]
[ConnectedInactive] --ResumeRequest--> [Connected]
[ConnectedInactive] --ReestablishmentRequest--> [Connected]
[ConnectedInactive] --DetachRRCRelease--> [Releasing]
[ConnectedInactive] --ConnectionFailure--> [Idle]
[Releasing] --DetachRRCRelease--> [Idle]
[Releasing] --ConnectionFailure--> [Idle]
```

**States**:
- **Idle**: UE not connected, no RRC context
- **Connected**: UE connected, active RRC context
- **ConnectedInactive**: UE in inactive state, context maintained
- **Releasing**: UE context being released

---

## Protocol Message Mapping

### NGAP Procedures

| Procedure | Initiating Message | Successful Outcome | Unsuccessful Outcome |
|-----------|-------------------|-------------------|---------------------|
| NG Setup | NGSetupRequest | NGSetupResponse | NGSetupFailure |
| Initial UE Message | InitialUEMessage | - | - |
| Downlink NAS Transport | DownlinkNASTransport | - | - |
| Uplink NAS Transport | UplinkNASTransport | - | - |
| Initial Context Setup | InitialContextSetupRequest | InitialContextSetupResponse | InitialContextSetupFailure |
| UE Context Release | UEContextReleaseCommand | UEContextReleaseComplete | - |
| PDU Session Resource Setup | PDUSessionResourceSetupRequest | PDUSessionResourceSetupResponse | PDUSessionResourceSetupFailure |
| Paging | Paging | - | - |

### F1AP Procedures

| Procedure | Initiating Message | Successful Outcome | Unsuccessful Outcome |
|-----------|-------------------|-------------------|---------------------|
| F1 Setup | F1SetupRequest | F1SetupResponse | F1SetupFailure |
| Initial UL RRC Message Transfer | InitialULRRCMessageTransfer | - | - |
| UL RRC Message Transfer | ULRRCMessageTransfer | - | - |
| DL RRC Message Transfer | DLRRCMessageTransfer | - | - |
| UE Context Setup | UEContextSetupRequest | UEContextSetupResponse | UEContextSetupFailure |
| UE Context Release | UEContextReleaseCommand | UEContextReleaseComplete | - |
| UE Context Modification | UEContextModificationRequest | UEContextModificationResponse | UEContextModificationFailure |
| Reset | Reset | ResetAcknowledge | - |
| gNB-DU Configuration Update | GNBDUConfigurationUpdate | GNBDUConfigurationUpdateAcknowledge | GNBDUConfigurationUpdateFailure |

### RRC Messages (carried in F1AP)

| Message Type | Direction | Purpose |
|-------------|-----------|---------|
| RRCSetupRequest | UL | Initial connection request |
| RRCSetup | DL | Connection establishment |
| RRCSetupComplete | UL | Connection establishment complete |
| RRCReconfiguration | DL | DRB setup, security activation |
| RRCReconfigurationComplete | UL | Reconfiguration complete |
| SecurityModeCommand | DL | AS security activation |
| SecurityModeComplete | UL | Security activation complete |
| UECapabilityEnquiry | DL | Request UE capabilities |
| UECapabilityInformation | UL | UE capabilities |
| RRCRelease | DL | Connection release |
| RRCReleaseComplete | UL | Release complete |
| RRCReestablishmentRequest | UL | Connection reestablishment |
| RRCReestablishment | DL | Reestablishment response |
| RRCReestablishmentComplete | UL | Reestablishment complete |

### NAS Messages (carried in NGAP)

| Message Type | Direction | Purpose |
|-------------|-----------|---------|
| Registration Request | UL | Initial registration |
| Registration Accept | DL | Registration accepted |
| Registration Complete | UL | Registration complete |
| Service Request | UL | Service request |
| Authentication Request | DL | Authentication challenge |
| Authentication Response | UL | Authentication response |
| Security Mode Command | DL | NAS security activation |
| Security Mode Complete | UL | Security activation complete |
| PDU Session Establishment Request | UL | PDU session setup |
| PDU Session Establishment Accept | DL | PDU session accepted |
| PDU Session Establishment Complete | UL | PDU session complete |

---

## Implementation Status

### Completed
- ✅ CU-CP context structure
- ✅ DU context and F1 Setup Request/Response handling
- ✅ AMF context and NG Setup Request/Response handling
- ✅ SCTP client (NGAP) and server (F1AP) implementations
- ✅ F1AP message decode/encode using library
- ✅ Configuration loading and validation
- ✅ Application lifecycle management
- ✅ RRC state machine definition

### In Progress / To Be Implemented
- ⏳ UE context full implementation (security, PDU sessions, DRBs)
- ⏳ Initial UE access flow (RRC Setup → Initial UE Message)
- ⏳ Security mode command procedure
- ⏳ UE capability enquiry
- ⏳ PDU session setup/modify/release
- ⏳ UE context release procedure
- ⏳ RRC reconfiguration procedure
- ⏳ RRC reestablishment
- ⏳ Handover procedures
- ⏳ Paging procedure
- ⏳ E1AP implementation (CU-UP interface)
- ⏳ Timer wheel for inactivity/paging timers
- ⏳ Metrics and observability

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    CU-CP Application                        │
│                    (internal/app)                           │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼──────┐ ┌──────▼──────┐ ┌─────▼──────┐
│   NGAP       │ │    F1AP      │ │   E1AP     │
│  Transport   │ │  Transport   │ │ Transport  │
│  (Client)    │ │  (Server)    │ │  (Server)  │
└───────┬──────┘ └──────┬──────┘ └─────┬──────┘
        │               │               │
        │               │               │
┌───────▼──────┐ ┌──────▼──────┐ ┌─────▼──────┐
│     AMF      │ │      DU     │ │   CU-UP    │
│   (NGAP)     │ │    (F1AP)   │ │   (E1AP)   │
└──────────────┘ └─────────────┘ └────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Context Stores                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ UE Pool  │  │ DU Pool  │  │AMF Pool │  │CU-UP Pool│    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Logic Engine                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ NGAP Handler │  │ F1AP Handler │  │E1AP Handler │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │         RRC State Machine (FSM)                     │  │
│  │  Idle ↔ Connected ↔ ConnectedInactive ↔ Releasing   │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Design Decisions

1. **Event-Driven Architecture**: Single-threaded logic loop processing events from transports, mirroring OAI's `rrc_gnb_task()`

2. **Protocol Library Integration**: Uses existing Go libraries for encode/decode, focusing on orchestration logic

3. **Context Storage**: Sharded maps (`sync.Map`) for scalability, with secondary indexes for efficient lookups

4. **SCTP Transport**: Separate client (NGAP) and server (F1AP/E1AP) implementations with connection pooling

5. **State Machine**: Generic FSM library for RRC state management

6. **Configuration**: YAML-based config loaded in application layer before service initialization

7. **Logging**: Structured logging with context fields (UE ID, DU ID, AMF ID, association ID)

---

## References

- OAI CU-CP Architecture: `docs/arch.md`
- Context Structures: `docs/context.md`
- UE Initial Procedure: `docs/ue_init_procedure.md`
- F1 Setup Procedure: `docs/f1setup.md`
- Protocol Libraries:
  - NGAP: `github.com/lvdund/ngap`
  - F1AP: `github.com/JocelynWS/f1-gen`
  - RRC: `github.com/lvdund/rrc`







-----------------------------------------------


Below is the **complete, step-by-step signaling** between **UE ↔ DU ↔ CU-CP ↔ AMF**, covering:

* **(1) gNB-DU, gNB-CU, AMF initial setup**
* **(2) UE power-on, RRC connection establishment**
* **(3) UE Registration (5GMM NAS)**
* **(4) UE Deregistration (5GMM NAS)**

---

## ✔ 1. INITIAL NETWORK BOOT ORDER

This answers: *“init connection DU, CU, AMF — which occur first?”*

#### **Step 1 — CU-CP boots and connects to AMF (NGAP over SCTP)**

1. CU opens SCTP association to AMF
2. CU sends **NG RESET** or AMF sends **NG SETUP REQUEST**
3. CU responds with **NG SETUP RESPONSE**

📘 **Source**: NGAP tutorial – NG SETUP is required before any UE procedure


---

#### **Step 2 — DU boots and connects to CU (F1AP over SCTP)**

1. DU opens SCTP association to CU
2. DU sends **F1 SETUP REQUEST**
3. CU responds with **F1 SETUP RESPONSE**

📘 **Source**: 3GPP TS 38.473 – F1 Setup is mandatory


---

#### **Step 3 — System ready to receive UEs**

After NG-Setup and F1-Setup, the topology is ready:

```
UE ---RRC--- DU ---F1--- CU ---NGAP--- AMF
```

---

## ✔ 2. UE POWER-ON → RRC CONNECTION SETUP

This answers: *“UE on → init UE connection”*

Procedure is defined in 3GPP TS 38.331 (RRC).

#### **(1) UE → DU : RRCSetupRequest**

* Contains establishment cause (mo-data, mo-signaling, etc.)

#### **(2) DU → CU : Initial UL RRC Message Transfer (F1AP)**

Carries the RRCSetupRequest inside **RRC Container**.

#### **(3) CU → DU : DL RRC Message Transfer**

Carries **RRCSetup**.

#### **(4) DU → UE : RRCSetup**

#### **(5) UE → DU : RRCSetupComplete**

Contains NAS: **Registration Request** (5GMM)

#### **(6) DU → CU : UL RRC Message Transfer**

Contains NAS Registration Request.

📘 **Source (tutorial on NAS/NGAP signaling)**

---

## ✔ 3. UE REGISTRATION PROCEDURE

This answers: *“UE executes Registration Request NAS procedure”*

Registration = NAS + NGAP + RRC tunneling.

#### **High-level flow**

```
UE → DU (RRC/NAS) → CU (RRC/NAS) → AMF (NGAP/NAS)
```

---

### **Detailed steps (complete signaling chain)**

#### **(1) UE sends Registration Request (NAS 5GMM)**

Sent inside RRCSetupComplete.

#### **(2) DU → CU : UL RRC Message Transfer (F1AP)**

Contains NAS.

#### **(3) CU → AMF : Initial UE Message (NGAP)**

Contains NAS Registration Request.

#### **(4) AMF → CU : Downlink NAS Transport (Authentication Request)**

#### **(5) CU → DU : DL RRC Message Transfer**

#### **(6) DU → UE : Authentication Request (NAS)**

#### **(7) UE → DU : Authentication Response (NAS)**

Relayed via DU → CU → AMF

#### **(8) AMF starts Security Mode Command**

Via NGAP → CU → DU → UE.

#### **(9) UE → Security Mode Complete**

Relayed DU → CU → AMF.

#### **(10) AMF → Initial Context Setup Request (NGAP)**

Sent to CU.
CU sends to DU via RRC/NAS containers.

📌 UE now gets **RRCReconfiguration** from CU (PDCP/RLC/SRB setup).

#### **(11) UE → RRCReconfigurationComplete → DU → CU**

#### **(12) AMF → Registration Accept → UE**

UE is now **Registered + Context active**.

📘 **Source**: NAS/NGAP registration tutorial

---

## ✔ 4. UE DEREGISTRATION PROCEDURE

This answers: *“UE executes De-registration request procedure”*

Two types:

* UE-initiated deregistration
* Network-initiated deregistration

Here is UE-initiated:

---

### **Detailed signaling steps**

#### **(1) UE → DU : RRC ULInformationTransfer**

Carries NAS: **Deregistration Request (UE initiated)**

#### **(2) DU → CU : UL RRC Message Transfer**

Contains NAS.

#### **(3) CU → AMF : Uplink NAS Transport (NGAP)**

#### **(4) AMF processes request and sends: Deregistration Accept**

Sent via NGAP → CU → DU → UE.

#### **(5) UE releases RRC connection**

Sends RRC ULInformationTransfer or directly RRCReleaseComplete.

#### **(6) CU → DU : UE Context Release Command (NGAP)**

#### **(7) DU → CU : UE Context Release Complete**


---

## ✔ COMPLETE SUMMARY (ordered by time)

### **A. Network boot**

1. CU ↔ AMF: NGAP setup
2. DU ↔ CU: F1AP setup

---

## **B. UE attaches (initial attach)**

1. RRCSetupRequest
2. Initial UL RRC Message Transfer
3. RRCSetup
4. RRCSetupComplete (with NAS)
5. Registration → Authentication → Security Mode
6. Initial Context Setup
7. RRCReconfiguration
8. Registration Complete

---

## **C. UE deregisters**

1. ULInformationTransfer (NAS Dereg Request)
2. Uplink NAS Transport (NGAP)
3. Dereg Accept → UE
4. UE Context Release

---

# ✔ 5. Official 3GPP documents for each step

| Procedure                         | 3GPP Spec     |
| --------------------------------- | ------------- |
| RRC Setup                         | **TS 38.331** |
| F1AP Setup / UE Context           | **TS 38.473** |
| NGAP Setup / Initial UE Message   | **TS 38.413** |
| Registration / Deregistration NAS | **TS 24.501** |
| End-to-End signaling              | **TS 23.502** |

---



-----------------------------------------------