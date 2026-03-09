package uecontext

import (
	"github.com/lvdund/ngap/ies"
)

// PDU Session states
const (
	PDU_SESSION_INACTIVE uint8 = iota
	PDU_SESSION_ESTABLISHING
	PDU_SESSION_ACTIVE
	PDU_SESSION_MODIFYING
	PDU_SESSION_RELEASING
)

// PduSessionContext represents a PDU session associated with a UE
type PduSessionContext struct {
	// Session Identifiers
	PduSessionId uint8 // PDU Session ID (1-15)
	State        uint8 // PDU_SESSION_*

	// QoS and Bearer Information
	Snssai   *ies.SNSSAI       // S-NSSAI for this session
	Dnn      string            // Data Network Name
	QosFlows []*QosFlowContext // List of QoS flows in this session

	// Data Radio Bearer mapping
	DrbId uint8 // DRB ID assigned to this session

	// GTP Tunnel Information (for reference - actual tunneling out of scope)
	// These would come from CU-UP via E1AP
	UlTeid uint32 // Uplink GTP TEID
	DlTeid uint32 // Downlink GTP TEID

	// NAS PDU
	NasPduSessionAccept []byte // PDU Session Establishment Accept NAS PDU
}

// QosFlowContext represents a QoS flow within a PDU session
type QosFlowContext struct {
	QosFlowId uint8 // QoS Flow Identifier (0-63)
	Qfi       uint8 // QoS Flow Identifier (same as above)
	FiveQi    int64 // 5QI value (e.g., 9 for default, 1 for voice)
	Priority  uint8 // Allocation and Retention Priority

	// QoS parameters (simplified - full structure in NGAP spec)
	// TODO: Add GBR (Guaranteed Bit Rate) parameters if needed
}
