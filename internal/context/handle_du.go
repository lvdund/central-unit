package context

import (
	f1ap "github.com/JocelynWS/f1-gen"
	"github.com/JocelynWS/f1-gen/ies"
	"github.com/lvdund/ngap/aper"
)

func (cu *CuCpContext) dispatchF1(rawMsg []byte) {
	if len(rawMsg) == 0 {
		cu.Error("F1AP message is empty")
		return
	}

	// Decode F1AP message
	pdu, err, _ := f1ap.F1apDecode(rawMsg)
	if err != nil {
		cu.Error("Error decoding F1AP message from DU: %v", err.Error())
		return
	}

	if pdu.Present == 0 || pdu.Message.ProcedureCode.Value == 0 {
		cu.Warn("Decoded F1AP PDU is nil or has no message")
		return
	}

	switch pdu.Present {
	case ies.F1apPduInitiatingMessage:
		switch pdu.Message.ProcedureCode.Value {
		case ies.ProcedureCode_F1Setup:
			cu.Info("Receive F1 Setup Request from DU")
			cu.handleF1SetupRequest(pdu.Message.Msg.(*ies.F1SetupRequest))
		case ies.ProcedureCode_InitialULRRCMessageTransfer:
			cu.Info("Receive Initial UL RRC Message from DU")
			if initialULMsg, ok := pdu.Message.Msg.(*ies.InitialULRRCMessageTransfer); ok {
				cu.handleInitialULRRCMessageTransfer(initialULMsg)
			} else {
				cu.Error("Failed to cast Initial UL RRC Message Transfer")
			}
		case ies.ProcedureCode_ULRRCMessageTransfer:
			cu.Info("Receive UL RRC Message from DU")
			if ulMsg, ok := pdu.Message.Msg.(*ies.ULRRCMessageTransfer); ok {
				cu.handleULRRCMessageTransfer(ulMsg)
			} else {
				cu.Error("Failed to cast Initial UL RRC Message Transfer")
			}
		default:
			cu.Warn("Received unknown F1AP message with procedure code %d", pdu.Message.ProcedureCode)
		}

	case ies.F1apPduSuccessfulOutcome:
		switch pdu.Message.ProcedureCode.Value {
		case ies.ProcedureCode_UEContextSetup:
			cu.Info("Receive UE Context Setup Response from DU")
			if ueContextSetupResponse, ok := pdu.Message.Msg.(*ies.UEContextSetupResponse); ok {
				cu.handleRRCUEContextSetupResponse(ueContextSetupResponse)
			} else {
				cu.Error("Failed to cast UE Context Setup Response")
			}
		}

	case ies.F1apPduUnsuccessfulOutcome:

	default:
		cu.Warn("Received F1AP message with unknown present type %d", pdu.Present)
	}
}

// plmnMatches checks if two PLMN byte arrays match
func (cu *CuCpContext) plmnMatches(plmn1, plmn2 []byte) bool {
	if len(plmn1) != len(plmn2) || len(plmn1) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		if plmn1[i] != plmn2[i] {
			return false
		}
	}
	return true
}

// cellIDsMatch checks if two cell IDs match (comparing bit strings)
func (cu *CuCpContext) cellIDsMatch(cellID1 aper.BitString, cellID2 uint64) bool {
	// Extract cell ID value from bit string and compare
	value1 := cu.extractCellIDValue(cellID1)
	return value1 == cellID2
}

// extractCellIDValue extracts the cell ID value from a bit string
func (cu *CuCpContext) extractCellIDValue(cellID aper.BitString) uint64 {
	// NRCellIdentity is 36 bits, extract as uint64
	if len(cellID.Bytes) == 0 {
		return 0
	}
	var value uint64
	bitsToRead := int(cellID.NumBits)
	if bitsToRead > 64 {
		bitsToRead = 64
	}
	for i := 0; i < len(cellID.Bytes) && bitsToRead > 0; i++ {
		bits := 8
		if bitsToRead < 8 {
			bits = bitsToRead
		}
		value = (value << uint(bits)) | uint64(cellID.Bytes[i]>>(8-bits))
		bitsToRead -= bits
	}
	return value
}
