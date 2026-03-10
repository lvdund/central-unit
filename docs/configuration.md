# Configuration Reference

## Configuration File

The CU-CP is configured via a YAML configuration file. The default path is `config/config.yml`, configurable via the `-config` command-line flag.

### File Format

```yaml
cucp:
  node_id: "0001"
  node_name: "gNB-CU-CP"
  plmn:
    mcc: "999"
    mnc: "70"
    mnc_length: 2
  slices:
    - sst: "01"
      sd: "010203"
  tac: "000001"

f1ap:
  local_address: "192.168.1.10"
  local_port: 38472
  sctp:
    in_streams: 2
    out_streams: 2
  timers:
    f1_setup_timer: "10s"

e1ap:
  local_address: "192.168.1.10"
  local_port: 38462
  sctp:
    in_streams: 2
    out_streams: 2

ngap:
  gnb_id: "000001"
  amf_address: "192.168.1.15"
  amf_port: 38412
  local_address: "192.168.1.10"
  local_port: 9487
  sctp:
    in_streams: 2
    out_streams: 2

logging:
  level: "info"
  format: "json"

features:
  split_architecture: true
  connected_inactive: false

tunables:
  ue_store_shards: 64
```

## Parameter Reference

### CU-CP Identity (`cucp`)

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `node_id` | string | Yes | Unique CU-CP node identifier (hex string) |
| `node_name` | string | Yes | Human-readable CU-CP name |
| `plmn.mcc` | string | Yes | Mobile Country Code (3 digits) |
| `plmn.mnc` | string | Yes | Mobile Network Code (2-3 digits) |
| `plmn.mnc_length` | integer | Yes | MNC length (2 or 3) |
| `slices[]` | array | No | Supported network slices (S-NSSAI) |
| `slices[].sst` | string | Yes | Slice/Service Type (hex) |
| `slices[].sd` | string | No | Slice Differentiator (hex) |
| `tac` | string | Yes | Tracking Area Code (hex) |

**PLMN Configuration:**

The PLMN (Public Land Mobile Network) configuration must match the core network and DU configurations:

- `mcc`: 3-digit Mobile Country Code (e.g., "999" for test networks)
- `mnc`: 2 or 3 digit Mobile Network Code
- `mnc_length`: Must be 2 or 3, matching the actual MNC length

**Slice Configuration:**

Network slices are defined via S-NSSAI (Single Network Slice Selection Assistance Information):

- `sst`: Standardized Slice/Service Type (1 = eMBB, 2 = URLLC, 3 = MIoT)
- `sd`: Slice Differentiator for operator-specific slices (optional)

### F1AP Interface (`f1ap`)

The F1AP interface connects the CU-CP to Distributed Units (DUs).

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `local_address` | string | Yes | - | Local IP for F1AP server |
| `local_port` | integer | Yes | - | Local SCTP port (3GPP: 38472) |
| `sctp.in_streams` | integer | Yes | - | Inbound SCTP streams |
| `sctp.out_streams` | integer | Yes | - | Outbound SCTP streams |
| `timers.f1_setup_timer` | duration | Yes | - | F1 Setup response timeout |

**Port Assignment:**

Per 3GPP TS 38.472, the F1-C interface uses SCTP port **38472**.

**SCTP Stream Configuration:**

Standard configuration uses 2 inbound and 2 outbound streams. Adjust based on expected connection load.

### E1AP Interface (`e1ap`)

The E1AP interface connects the CU-CP to the CU-UP (User Plane).

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `local_address` | string | Yes | - | Local IP for E1AP server |
| `local_port` | integer | Yes | - | Local SCTP port (3GPP: 38462) |
| `sctp.in_streams` | integer | Yes | - | Inbound SCTP streams |
| `sctp.out_streams` | integer | Yes | - | Outbound SCTP streams |

**Port Assignment:**

Per 3GPP TS 38.462, the E1 interface uses SCTP port **38462**.

**Implementation Status:** E1AP message handling is not yet implemented.

### NGAP Interface (`ngap`)

The NGAP interface connects the CU-CP to the AMF (Access and Mobility Management Function).

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `gnb_id` | string | Yes | - | gNB identifier (hex) |
| `amf_address` | string | Yes | - | AMF IP address |
| `amf_port` | integer | Yes | - | AMF SCTP port (3GPP: 38412) |
| `local_address` | string | Yes | - | Local IP for NGAP client |
| `local_port` | integer | Yes | - | Local SCTP port |
| `sctp.in_streams` | integer | Yes | - | Inbound SCTP streams |
| `sctp.out_streams` | integer | Yes | - | Outbound SCTP streams |

**Port Assignment:**

Per 3GPP TS 38.412, the N2 interface uses SCTP port **38412**.

**gNB Identification:**

The `gnb_id` parameter identifies this gNB within the PLMN. Format is a hex string representing the gNB ID (22-32 bits).

### Logging (`logging`)

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `level` | string | Yes | "info" | Log verbosity level |
| `format` | string | Yes | "json" | Output format |

**Log Levels:**

| Level | Description |
|-------|-------------|
| `trace` | Detailed protocol tracing |
| `debug` | Debug information |
| `info` | Operational messages (recommended) |
| `warn` | Warning conditions |
| `error` | Error conditions |
| `fatal` | Fatal errors (terminates) |

**Output Formats:**

| Format | Use Case |
|--------|----------|
| `json` | Production (structured logging) |
| `text` | Development (human-readable) |

### Feature Flags (`features`)

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `split_architecture` | boolean | false | Enable CU/DU split mode |
| `connected_inactive` | boolean | false | Enable RRC Inactive state support |

### Tunables (`tunables`)

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ue_store_shards` | integer | 64 | UE context hash map shards |

**UE Store Sharding:**

The `ue_store_shards` parameter controls the number of shards in the UE context map. Higher values reduce lock contention in multi-threaded scenarios (when implemented).

## Validation

The configuration loader performs the following validations:

1. **Required Fields**: All required parameters must be present
2. **PLMN Format**: MCC must be 3 digits, MNC length must be 2 or 3
3. **SCTP Streams**: `in_streams` and `out_streams` must be non-zero
4. **Endpoints**: All addresses and ports must be specified
5. **Logging Format**: Must be "json" or "text"
6. **Timer Values**: Duration strings must be parseable (e.g., "10s", "1m")

## Environment-Specific Configurations

### Development

```yaml
logging:
  level: "debug"
  format: "text"
```

### Production

```yaml
logging:
  level: "info"
  format: "json"

tunables:
  ue_store_shards: 128
```

### Multi-DU Deployment

For deployments with multiple DUs, increase SCTP streams:

```yaml
f1ap:
  sctp:
    in_streams: 4
    out_streams: 4
```
