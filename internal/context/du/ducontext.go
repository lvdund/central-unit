package du

import (
	"fmt"

	"github.com/JocelynWS/f1-gen/ies"
	"github.com/ishidawataru/sctp"
)

// DU main states
const (
	DU_INACTIVE string = "DU_INACTIVE"
	DU_ACTIVE   string = "DU_ACTIVE"
	DU_LOST     string = "DU_LOST"
)

// GNBDU represents a Distributed Unit (DU) context
// Based on nr_rrc_du_container_t from OAI
type GNBDU struct {
	DuId        int64               // DU ID (GNB-DU-ID)
	DuName      string              // DU name
	State       string              // DU state (INACTIVE, ACTIVE, LOST)
	Tnla        TNLAssociation      // Transport Network Layer Association
	SetupReq    *ies.F1SetupRequest // F1 Setup Request message
	MIB         []byte              // Decoded Master Information Block (raw bytes for now)
	SIB1        []byte              // Decoded System Information Block Type 1 (raw bytes for now)
	MTC         []byte              // Decoded Measurement Timing Configuration (raw bytes for now)
	ServedCells []ServedCell        // List of served cells
}

// TNLAssociation represents the transport network layer association
type TNLAssociation struct {
	SctpConn         *sctp.SCTPConn // Raw SCTP connection for DU (incoming connections)
	TnlaWeightFactor int64
	Streams          uint16
}

// ServedCell represents a cell served by the DU
type ServedCell struct {
	CellID       uint64 // Physical Cell ID
	GlobalCellID string // Global Cell Identifier
	PLMN         PLMNInfo
	PCI          uint16 // Physical Cell Identifier
	BandwidthMHz uint16
	TAC          []byte // Tracking Area Code
}

// PLMNInfo represents PLMN information
type PLMNInfo struct {
	MCC string
	MNC string
}

// SendF1ap sends F1AP message to the DU
func (du *GNBDU) SendF1ap(pdu []byte) error {
	if du.Tnla.SctpConn == nil {
		return fmt.Errorf("SCTP connection not established for DU %d", du.DuId)
	}
	info := &sctp.SndRcvInfo{
		PPID:   60, // F1AP PPID
		Stream: 0,
	}
	_, err := du.Tnla.SctpConn.SCTPWrite(pdu, info)
	return err
}

// GetCellByID returns served cell information by cell ID
func (du *GNBDU) GetCellByID(cellID uint64) *ServedCell {
	for i := range du.ServedCells {
		if du.ServedCells[i].CellID == cellID {
			return &du.ServedCells[i]
		}
	}
	return nil
}

// IsActive returns true if DU is in active state
func (du *GNBDU) IsActive() bool {
	return du.State == DU_ACTIVE
}
