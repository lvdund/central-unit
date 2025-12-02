package context

import (
	"central-unit/internal/common/logger"
	"central-unit/internal/context/du"
	"fmt"

	"github.com/JocelynWS/f1-gen/ies"
	"github.com/lvdund/rrc"
	rrcies "github.com/lvdund/rrc/ies"
)

// Based on rrc_gNB_process_f1_setup_req from OAI
func (cu *CuCpContext) handleF1SetupRequest(setupReq *ies.F1SetupRequest) {
	transactionID := setupReq.TransactionID
	duId := setupReq.GNBDUID
	duName := string(setupReq.GNBDUName)
	cellItem := setupReq.GNBDUServedCellsList[0]
	cellInfo := cellItem.ServedCellInformation
	cellID := cellInfo.NRCGI.NRCellIdentity
	pci := int64(cellInfo.NRPCI.Value)
	mtc := cellInfo.MeasurementTimingConfiguration
	cu.Info("Received F1 Setup Request from gNB_DU %d (%s)", duId, duName)

	numCells := len(setupReq.GNBDUServedCellsList)
	if numCells != 1 {
		cu.Error("can only handle one DU cell, but gNB_DU %d has %d", duId, numCells)
		return
	}

	if len(cellInfo.ServedPLMNs) == 0 {
		cu.Error("No PLMN in served cell information")
		return
	}
	cellPLMNBytes := cellInfo.ServedPLMNs[0].PLMNIdentity
	// 10011001 11111001 00110111
	// 10011001 11111001 00000111

	cuPLMNBytes := cu.GetMccAndMncInOctets()
	if !cu.plmnMatches(cellPLMNBytes, cuPLMNBytes) {
		cu.Error("PLMN mismatch: CU %s.%s, DU %.8b", cu.ControlInfo.mcc, cu.ControlInfo.mnc, cuPLMNBytes)
		return
	}

	// existingDUs := make(map[int64]*du.GNBDU)
	// cu.DuPool.Range(func(key, value any) bool {
	// 	if d, ok := value.(*du.GNBDU); ok {
	// 		existingDUs[d.DuId] = d
	// 	}
	// 	return true
	// })
	// if existingDU, exists := existingDUs[duId]; exists {
	// 	cu.Error("gNB-DU ID: existing DU %s already has ID %d, rejecting requesting gNB-DU",
	// 		existingDU.DuName, duId)
	// 	return
	// }

	// for _, existingDU := range existingDUs {
	// 	if len(existingDU.ServedCells) > 0 {
	// 		existingCell := existingDU.ServedCells[0]
	// 		if cu.cellIDsMatch(cellID, existingCell.CellID) || int64(existingCell.PCI) == pci {
	// 			cu.Error("existing DU %s on already has cellID %d/physCellId %d, rejecting requesting gNB-DU with cellID %d/physCellId %d",
	// 				existingDU.DuName, existingCell.CellID, existingCell.PCI, cellID, pci)
	// 			return
	// 		}
	// 	}
	// }

	var mib []byte
	var sib1 []byte
	if cellItem.GNBDUSystemInformation != nil {
		mib = cellItem.GNBDUSystemInformation.MIBMessage
		sib1 = cellItem.GNBDUSystemInformation.SIB1Message
	}

	cu.Info("Accepting DU %d (%s), sending F1 Setup Response", duId, duName)
	cu.Info("DU uses RRC version %x", setupReq.GNBDURRCVersion.LatestRRCVersion.Bytes)

	var duCtx *du.GNBDU = &du.GNBDU{}
	duCtx.Logger = logger.InitLogger("info", map[string]string{"mod": "du"})
	duCtx.DuId = duId
	duCtx.DuName = duName
	duCtx.State = du.DU_ACTIVE
	duCtx.SetupReq = setupReq
	duCtx.MIB = mib
	duCtx.SIB1 = sib1
	duCtx.MTC = mtc
	cellIDValue := cu.extractCellIDValue(cellID)
	duCtx.ServedCells = []du.ServedCell{
		{
			CellID:       cellIDValue,
			GlobalCellID: fmt.Sprintf("%x-%x", cellPLMNBytes, cellIDValue),
			PLMN: du.PLMNInfo{
				MCC: cu.ControlInfo.mcc,
				MNC: cu.ControlInfo.mnc,
			},
			PCI: uint16(pci),
			TAC: cellInfo.FiveGSTAC,
		},
	}

	cu.DuPool.Store(duId, duCtx)
	cu.Info("==== Store DU %d ====", duId)
	cu.DU = duCtx

	// Create cell to activate
	cellToActivate := ies.CellstobeActivatedListItem{
		NRCGI: ies.NRCGI{
			PLMNIdentity:   cellPLMNBytes,
			NRCellIdentity: cellID,
		},
		NRPCI: &ies.NRPCI{
			Value: pci,
		},
	}

	cu.Info("Create F1 SetupResponse")
	if err := duCtx.SendF1SetupResponse(transactionID, setupReq.GNBDURRCVersion, []ies.CellstobeActivatedListItem{cellToActivate}, cu.TempDuConn); err != nil {
		cu.Error("Error sending F1 Setup Response: %v", err)
	} else {
		cu.Info("F1 Setup Procedure successfully with DU %d (%s)", duCtx.DuId, duCtx.DuName)
	}
}

func (cu *CuCpContext) handleInitialULRRCMessageTransfer(msg *ies.InitialULRRCMessageTransfer) {
	cu.Info("Processing Initial UL RRC Message Transfer: DU-UE-ID=%d, C-RNTI=%d", msg.GNBDUUEF1APID, msg.CRNTI)

	// duCtx, ok := cu.DuPool.Load(msg.GNBDUUEF1APID)
	// if !ok {
	// 	cu.Error("DU %d not found in DuPool", msg.GNBDUUEF1APID)
	// 	return
	// }
	duCtx := cu.DU

	ulCcchMsg := rrcies.UL_CCCH_Message{}
	err := rrc.Decode(msg.RRCContainer, &ulCcchMsg)
	if err != nil {
		cu.Error("Error decoding RRC container: %s", err.Error())
		return
	}
	rrcSetupRequest := ulCcchMsg.Message.C1.RrcSetupRequest
	if err := cu.handleRRCSetupRequest(
		duCtx,
		&rrcSetupRequest.RrcSetupRequest,
		msg,
	); err != nil {
		cu.Error("Error handling RRC Setup Request: %s", err.Error())
	}

	//WARN: now only support RrcSetupRequest
	//TODO: RRCResumeRequest, RRCReestablishmentRequest, RRCSystemInfoRequest
}

func (cu *CuCpContext) handleULRRCMessageTransfer(msg *ies.ULRRCMessageTransfer) {
	var err error

	// // Validation check 1: msg.GNBCUUEF1APID must exist in cu.RrcUePool
	// ueValue, exists := cu.RrcUePool.Load(msg.GNBCUUEF1APID)
	// if !exists {
	// 	cu.Error("UL RRC Message Transfer: CU UE ID %d not found in RrcUePool",
	// 		msg.GNBCUUEF1APID)
	// 	return
	// }

	// // Validation check 2: Load UE and verify msg.GNBDUUEF1APID == ue.DuUeId
	// ue, ok := ueValue.(*uecontext.GNBUe)
	// if !ok {
	// 	cu.Error("UL RRC Message Transfer: Invalid UE type in RrcUePool for CU UE ID %d",
	// 		msg.GNBCUUEF1APID)
	// 	return
	// }

	ue := cu.UE

	if uint64(msg.GNBDUUEF1APID) != ue.DuUeId {
		cu.Error("UL RRC Message Transfer: DU UE ID mismatch. Expected %d, got %d",
			ue.DuUeId, msg.GNBDUUEF1APID)
		return
	}

	// Validation check 3: msg.SRBID must be >= 1
	if msg.SRBID < 1 {
		cu.Error("UL RRC Message Transfer: Invalid SRBID %d, must be >= 1", msg.SRBID)
		return
	}

	ulDcchMsg := rrcies.UL_DCCH_Message{}
	err = rrc.Decode(msg.RRCContainer, &ulDcchMsg)
	if err != nil {
		cu.Error("Err decode RRC from UL RRC Message Transfer: %s - %v", err.Error(), msg.RRCContainer)
		return
	}

	// Check if message uses C1 choice
	if ulDcchMsg.Message.Choice != 1 { // 1 = C1, other values are MessageClassExtension
		cu.Error("UL RRC Message Transfer: Unsupported message choice %d", ulDcchMsg.Message.Choice)
		return
	}

	if ulDcchMsg.Message.C1 == nil {
		cu.Error("UL RRC Message Transfer: C1 is nil")
		return
	}

	// Switch on C1 message type
	switch ulDcchMsg.Message.C1.Choice {
	case rrcies.UL_DCCH_MessageType_C1_Choice_RrcSetupComplete:
		// Handle RRC Setup Complete
		if ulDcchMsg.Message.C1.RrcSetupComplete == nil {
			cu.Error("UL RRC Message Transfer: RrcSetupComplete is nil")
			return
		}
		if err := cu.handleRrcSetupComplete(ue, ulDcchMsg.Message.C1.RrcSetupComplete); err != nil {
			cu.Error("Error handling RRC Setup Complete: %s", err.Error())
		}

	case rrcies.UL_DCCH_MessageType_C1_Choice_UlInformationTransfer:
		// Handle UL Information Transfer (carries NAS messages)
		if ulDcchMsg.Message.C1.UlInformationTransfer == nil {
			cu.Error("UL RRC Message Transfer: UlInformationTransfer is nil")
			return
		}
		if err := cu.handleULInformationTransfer(ue, ulDcchMsg.Message.C1.UlInformationTransfer); err != nil {
			cu.Error("Error handling UL Information Transfer: %s", err.Error())
		}

	case rrcies.UL_DCCH_MessageType_C1_Choice_SecurityModeComplete:
		// Handle Security Mode Complete
		if ulDcchMsg.Message.C1.SecurityModeComplete == nil {
			cu.Error("UL RRC Message Transfer: SecurityModeComplete is nil")
			return
		}
		if err := cu.handleRRCSecurityModeComplete(ue, ulDcchMsg.Message.C1.SecurityModeComplete); err != nil {
			cu.Error("Error handling Security Mode Complete: %s", err.Error())
		}
	case rrcies.UL_DCCH_MessageType_C1_Choice_RrcReconfigurationComplete:
		// Handle RRC Reconfiguration Complete
		if ulDcchMsg.Message.C1.RrcReconfigurationComplete == nil {
			cu.Error("UL RRC Message Transfer: RrcReconfigurationComplete is nil")
			return
		}
		if err := cu.handleRRCReconfigurationComplete(ue, ulDcchMsg.Message.C1.RrcReconfigurationComplete); err != nil {
			cu.Error("Error handling RRC Reconfiguration Complete: %s", err.Error())
		}
	default:
		cu.Warn("UL RRC Message Transfer: Unsupported C1 message type %d", ulDcchMsg.Message.C1.Choice)
	}
}

func (cu *CuCpContext) handleRRCUEContextSetupResponse(msg *ies.UEContextSetupResponse) {

	// ueValue, _ := cu.RrcUePool.Load(0)
	// ue, ok := ueValue.(*uecontext.GNBUe)
	// if !ok {
	// 	cu.Error("Error getting UE")
	// 	return
	// }
	ue := cu.UE

	masterCellGroupBytes, err := rrc.Encode(ue.MasterCellGroup)
	if err != nil {
		cu.Error("Error encoding MasterCellGroup: %s", err.Error())
		return
	}

	rrcmsg := rrcies.RRCReconfiguration{
		Rrc_TransactionIdentifier: rrcies.RRC_TransactionIdentifier{Value: 0},
		CriticalExtensions: rrcies.RRCReconfiguration_CriticalExtensions{
			Choice: rrcies.RRCReconfiguration_CriticalExtensions_Choice_RrcReconfiguration,
			RrcReconfiguration: &rrcies.RRCReconfiguration_IEs{
				RadioBearerConfig: &rrcies.RadioBearerConfig{
					Srb_ToAddModList: &rrcies.SRB_ToAddModList{
						Value: []rrcies.SRB_ToAddMod{
							{
								Srb_Identity: rrcies.SRB_Identity{Value: 2},
							},
						},
					},
					Drb_ToAddModList: &rrcies.DRB_ToAddModList{
						Value: []rrcies.DRB_ToAddMod{rrcies.DRB_ToAddMod{
							Drb_Identity: rrcies.DRB_Identity{Value: 0},
						}},
					},
				},
				SecondaryCellGroup: &masterCellGroupBytes,
				MeasConfig:         &rrcies.MeasConfig{},
				NonCriticalExtension: &rrcies.RRCReconfiguration_v1530_IEs{
					MasterCellGroup: &masterCellGroupBytes,
					DedicatedNAS_MessageList: []rrcies.DedicatedNAS_Message{rrcies.DedicatedNAS_Message{
						Value: ue.RegistrationAccept,
					}},
				},
			},
		},
	}

	dlDcchMsg := rrcies.DL_DCCH_Message{
		Message: rrcies.DL_DCCH_MessageType{
			Choice: rrcies.DL_DCCH_MessageType_Choice_C1,
			C1: &rrcies.DL_DCCH_MessageType_C1{
				Choice:             rrcies.DL_DCCH_MessageType_C1_Choice_RrcReconfiguration,
				RrcReconfiguration: &rrcmsg,
			},
		},
	}

	buf, err := rrc.Encode(&dlDcchMsg)
	if err != nil {
		cu.Error("Error encoding DL DCCH Message: %s", err.Error())
		return
	}

	// duCtx, _ := cu.DuPool.Load(ue.DuUeId)
	duCtx := cu.DU
	duCtx.SendF1ap(buf)
	cu.Info("RRC Reconfiguration sent successfully")
}
