# Development Roadmap

## Current Implementation Status

### Completed Features

| Feature | Status | Implementation |
|---------|--------|----------------|
| CU-CP Context Management | Complete | `internal/context/context_cucp.go` |
| NGAP Client (SCTP) | Complete | `internal/transport/sctpclient.go` |
| NGAP Message Handling | Complete | `internal/context/protocol_ngap.go` |
| F1AP Message Handling | Complete | `internal/context/protocol_f1c.go` |
| RRC Message Construction | Complete | `internal/context/protocol_rrc.go` |
| UE State Machine | Complete | `internal/context/uecontext/` |
| DU Context Management | Complete | `internal/context/du/` |
| AMF Context Management | Complete | `internal/context/amfcontext/` |
| Configuration System | Complete | `pkg/config/config.go` |
| FSM Framework | Complete | `internal/common/fsm/` |
| Milenage Authentication | Complete | `internal/context/uecontext/milenage.go` |

### Incomplete / Partial Features

| Feature | Status | Notes |
|---------|--------|-------|
| F1AP SCTP Server | Partial | Code exists in `sctpserver.go` but commented out |
| E1AP Implementation | Not Started | Interface defined, handlers not implemented |
| Context Mutex Protection | TODO | `context_cucp.go:142` - concurrent access not protected |
| RRC Resume | TODO | `protocol_f1c.go:151` |
| RRC Reestablishment | TODO | `protocol_f1c.go:151` |
| Security Context Derivation | TODO | `handle_amf.go:218,226,227` |

## Planned Features

### Phase 1: E1AP Implementation

Enable CU-CP ↔ CU-UP communication for bearer management:

- **E1 Setup Procedure**: Establish association with CU-UP
- **Bearer Context Setup**: Create QoS flows and DRBs
- **Bearer Context Modification**: Update QoS parameters
- **Bearer Context Release**: Tear down bearer resources

**Dependencies:**
- CU-UP implementation or simulator
- E1AP ASN.1 encoding library

### Phase 2: Multi-Connection Support

Scale to production-level connection management:

| Capability | Description |
|------------|-------------|
| Multiple DU Connections | Support N DUs with individual F1 associations |
| Multiple AMF Associations | Connect to multiple AMFs for redundancy |
| Connection Pooling | Reuse SCTP associations efficiently |
| Load Balancing | Distribute UEs across AMFs |

**Dependencies:**
- Context map mutex implementation
- Connection state tracking per endpoint

### Phase 3: Extended Procedures

Complete 3GPP procedure support:

#### Session Management

| Procedure | 3GPP Reference |
|-----------|----------------|
| PDU Session Establishment | TS 38.413 §8.3 |
| PDU Session Modification | TS 38.413 §8.4 |
| PDU Session Release | TS 38.413 §8.5 |

#### Mobility Management

| Procedure | 3GPP Reference |
|-----------|----------------|
| Intra-DU Handover | TS 38.401 |
| Inter-DU Handover | TS 38.401 |
| Inter-gNB Handover | TS 38.413 §8.9 |
| RRC Reestablishment | TS 38.331 §5.3.7 |
| RRC Resume | TS 38.331 §5.3.9 |

#### Paging

| Procedure | 3GPP Reference |
|-----------|----------------|
| Paging Request | TS 38.413 §8.7 |
| UE Paging | TS 38.331 §5.3.2 |

#### Context Management

| Procedure | 3GPP Reference |
|-----------|----------------|
| UE Context Release Request | TS 38.413 §8.2 |
| UE Context Release Complete | TS 38.413 §8.2 |
| UE Context Modification | TS 38.413 §8.6 |

### Phase 4: Security Context

Complete security functionality:

| Capability | Description |
|------------|-------------|
| 5G-AKA Authentication | Full authentication flow with AMF |
| Key Derivation | KgNB, KUPenc, KUPint, KRRcenc, KRRCint |
| Integrity Protection | RRC and NAS message integrity |
| Ciphering | RRC and NAS message encryption |
| Security Mode Command | Security capability negotiation |

**Dependencies:**
- Security algorithm implementations (AES, SNOW 3G, ZUC)
- Key hierarchy implementation per TS 33.501

## Known Issues

### Thread Safety

The `CuCpContext` maps (`ues`, `dus`, `amfs`) lack mutex protection. This is acceptable for single-threaded operation but will cause race conditions if parallel event processing is introduced.

**Resolution:** Implement sharded locking with per-shard mutexes.

### SCTP Server

The F1AP SCTP server implementation in `sctpserver.go` is fully coded but commented out. The current architecture requires integration with the context event loop.

**Resolution:** Uncomment and integrate with `f1_server.go` lifecycle management.

### Error Handling

Error propagation from protocol handlers to transport layer is incomplete. Some errors are logged but not properly signaled to the SCTP layer for connection cleanup.

**Resolution:** Implement structured error types with transport-layer callbacks.

## Contribution Guidelines

### Priority Areas

Contributions are particularly welcome in:

1. **E1AP Implementation**: Protocol handlers, message encoding
2. **Context Thread Safety**: Mutex implementation, sharding
3. **Testing**: Unit tests, integration tests, protocol conformance
4. **Documentation**: Architecture diagrams, sequence diagrams

### Code Standards

- Follow Go standard formatting (`gofmt`)
- Document all exported types and functions
- Maintain OAI ITTI naming conventions for compatibility
- Add TODO comments for incomplete features with issue references

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2026-01 | Initial implementation: NGAP, F1AP, RRC |
| - | - | Current: See git history |
