package du

import (
	"bytes"
	"fmt"

	f1ap "github.com/JocelynWS/f1-gen"
	"github.com/JocelynWS/f1-gen/ies"
)

// F1apDecode decodes F1AP PDU bytes and returns the decoded message
// Uses the F1AP library github.com/JocelynWS/f1-gen
func F1apDecode(data []byte) (*F1apPDU, error, interface{}) {
	pdu, err, diagnostics := f1ap.F1apDecode(data)
	if err != nil {
		return nil, err, diagnostics
	}

	// Convert library F1apPdu to our F1apPDU type
	result := &F1apPDU{
		Present: int(pdu.Present),
		Message: &F1apMessage{
			ProcedureCode: int64(pdu.Message.ProcedureCode.Value),
			Msg:           pdu.Message.Msg,
		},
	}

	return result, nil, diagnostics
}

// F1apPDU represents a decoded F1AP PDU
// Wrapper around f1ap.F1apPdu for easier use
type F1apPDU struct {
	Present int
	Message *F1apMessage
}

// F1apMessage represents the message part of F1AP PDU
// Wrapper around f1ap.F1apMessage for easier use
type F1apMessage struct {
	ProcedureCode int64
	Msg           interface{} // The actual decoded message (e.g., *ies.F1SetupRequest)
}

// Based on CU_send_F1_SETUP_RESPONSE from OAI
func (du *GNBDU) SendF1SetupResponse(transactionID int64, gnbCURRCVersion ies.RRCVersion, cellsToActivate []ies.CellstobeActivatedListItem) error {
	// Create F1 Setup Response message
	// RRCVersion uses 3 bits as shown in the test file (f1_test.go)
	msg := ies.F1SetupResponse{
		TransactionID:          transactionID,
		GNBCURRCVersion:        gnbCURRCVersion,
		CellstobeActivatedList: cellsToActivate,
		GNBCUName:              []byte("CU-CP"),
	}

	// Encode the message
	var buf bytes.Buffer
	if err := msg.Encode(&buf); err != nil {
		return fmt.Errorf("encode F1 Setup Response: %w", err)
	}

	// Send via SCTP
	return du.SendF1ap(buf.Bytes())
}

// SendF1SetupFailure sends F1 Setup Failure to the DU
func (du *GNBDU) SendF1SetupFailure(transactionID int64, cause interface{}) error {
	// Create F1 Setup Failure message
	msg := ies.F1SetupFailure{
		TransactionID: transactionID,
	}

	// Set cause if provided
	if cause != nil {
		if c, ok := cause.(ies.Cause); ok {
			msg.Cause = c
		} else {
			return fmt.Errorf("invalid cause type for F1 Setup Failure")
		}
	} else {
		// Default cause if not provided - use Misc/Unspecified
		msg.Cause = ies.Cause{
			Choice: 4, // Misc cause
			Misc: &ies.CauseMisc{
				Value: 0, // Unspecified
			},
		}
	}

	// Encode the message
	var buf bytes.Buffer
	if err := msg.Encode(&buf); err != nil {
		return fmt.Errorf("encode F1 Setup Failure: %w", err)
	}

	// Send via SCTP
	return du.SendF1ap(buf.Bytes())
}

// DefaultRRCVersion returns default RRC version bytes
func DefaultRRCVersion() []byte {
	return []byte{0x0c, 0x22, 0x38} // Default RRC version
}
