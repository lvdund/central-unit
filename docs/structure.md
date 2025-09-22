## Structure Central Unit
```
cucp/
├── main.go                        # Main entry point
├── config/
│   ├── config.go                  # Configuration management
│   └── types.go                   # Configuration types
├── pkg/
│   ├── core/
│   │   ├── app/                   # Application layer
│   │   │   ├── app.go             # Main application controller
│   │   │   └── coordinator.go     # Task coordination
│   │   ├── rrc/                   # RRC layer
│   │   │   ├── rrc.go             # Main RRC implementation
│   │   │   ├── ue_context.go      # UE context management
│   │   │   ├── mobility.go        # Handover management
│   │   │   ├── bearers.go         # Radio bearer management
│   │   │   └── security.go        # Security functions
│   │   ├── pdcp/                  # PDCP Control Plane
│   │   │   ├── pdcp.go            # PDCP entity
│   │   │   ├── security.go        # Ciphering/Integrity
│   │   │   └── manager.go         # PDCP manager
│   │   └── interfaces/            # Interface definitions
│   │       ├── f1ap.go            # F1AP interface
│   │       ├── e1ap.go            # E1AP interface
│   │       └── ngap.go            # NGAP interface
│   ├── protocols/
│   │   ├── f1ap/                  # F1AP implementation
│   │   │   ├── task.go            # F1AP task
│   │   │   ├── handlers.go        # Message handlers
│   │   │   ├── encoder.go         # ASN.1 encoding
│   │   │   ├── decoder.go         # ASN.1 decoding
│   │   │   ├── setup.go           # F1 Setup procedures
│   │   │   ├── ue_context.go      # UE context procedures
│   │   │   ├── rrc_transfer.go    # RRC message transfer
│   │   │   └── paging.go          # Paging procedures
│   │   ├── e1ap/                  # E1AP implementation
│   │   │   ├── task.go            # E1AP CU-CP task
│   │   │   ├── handlers.go        # Message handlers
│   │   │   ├── encoder.go         # ASN.1 encoding
│   │   │   ├── decoder.go         # ASN.1 decoding
│   │   │   ├── setup.go           # E1 Setup procedures
│   │   │   └── bearer_context.go  # Bearer context management
│   │   ├── ngap/                  # NGAP implementation
│   │   │   ├── task.go            # NGAP task
│   │   │   ├── handlers.go        # Message handlers
│   │   │   ├── encoder.go         # ASN.1 encoding
│   │   │   ├── decoder.go         # ASN.1 decoding
│   │   │   └── procedures.go      # NGAP procedures
│   │   └── sctp/                  # SCTP transport
│   │       ├── server.go          # SCTP server
│   │       ├── client.go          # SCTP client
│   │       └── connection.go      # Connection management
│   ├── messaging/
│   │   ├── itti.go                # Inter-task messaging
│   │   ├── dispatcher.go          # Message dispatcher
│   │   └── types.go               # Message types
│   └── utils/
│       ├── logger.go              # Logging utilities
│       ├── timer.go               # Timer management
│       └── crypto.go              # Cryptographic functions
└── test/
    ├── unit/                      # Unit tests
    └── integration/               # Integration tests
```

