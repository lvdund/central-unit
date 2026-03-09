package context

import (
	"central-unit/internal/context/amfcontext"
	"central-unit/internal/context/uecontext"
	"fmt"

	f1ap "github.com/JocelynWS/f1-gen"
	f1ies "github.com/JocelynWS/f1-gen/ies"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/rrc"
	rrcies "github.com/lvdund/rrc/ies"
)

func (cu *CuCpContext) handlePduSessionResourceSetupRequest(
	amf *amfcontext.GNBAmf,
	msg *ies.PDUSessionResourceSetupRequest,
) {
	cu.Info("Processing PDU Session Resource Setup Request")

	ue, err := cu.GetUEByNgapId(msg.RANUENGAPID)
	if err != nil {
		cu.Error("UE not found for RAN-UE-NGAP-ID %d: %v", msg.RANUENGAPID, err)
		return
	}

	ue.AmfUeNgapId = msg.AMFUENGAPID

	if msg.PDUSessionResourceSetupListSUReq == nil || len(msg.PDUSessionResourceSetupListSUReq) == 0 {
		cu.Error("PDUSessionResourceSetupListSUReq is empty")
		return
	}

	for _, item := range msg.PDUSessionResourceSetupListSUReq {
		cu.Info("Processing PDU Session ID: %d", item.PDUSessionID)

		pduSessionId := uint8(item.PDUSessionID)
		nasPdu := item.PDUSessionNASPDU

		if ue.PduSessions == nil {
			ue.PduSessions = make(map[uint8]*uecontext.PduSessionContext)
		}

		if _, exists := ue.PduSessions[pduSessionId]; exists {
			cu.Error("PDU Session ID %d already exists for UE", pduSessionId)
			continue
		}

		drbId := pduSessionId

		pduSession := &uecontext.PduSessionContext{
			PduSessionId:        pduSessionId,
			State:               uecontext.PDU_SESSION_ESTABLISHING,
			DrbId:               drbId,
			NasPduSessionAccept: nasPdu,
			UlTeid:              0x12345678,
			DlTeid:              0x87654321,
		}

		ue.PduSessions[pduSessionId] = pduSession
		ue.NumActiveSessions++

		cu.Info("Created PDU Session ID=%d, DRB ID=%d for UE RAN-NGAP-ID=%d",
			pduSessionId, drbId, ue.RanUeNgapId)

		cu.Info("TODO: E1AP Bearer Context Setup Request to CU-UP (out of scope)")

		err = cu.sendF1UEContextModificationRequest(ue, pduSession)
		if err != nil {
			cu.Error("Failed to send F1AP UE Context Modification Request: %v", err)
			continue
		}
	}

	cu.Info("PDU Session setup initiated, waiting for F1AP and RRC confirmation")
}

func (cu *CuCpContext) sendF1UEContextModificationRequest(
	ue *uecontext.GNBUe,
	pduSession *uecontext.PduSessionContext,
) error {
	cu.Info("Building F1AP UE Context Modification Request for PDU Session ID=%d", pduSession.PduSessionId)

	duCtx, err := cu.GetDUForUE(ue)
	if err != nil {
		return fmt.Errorf("DU not found for UE: %v", err)
	}

	msg := f1ies.UEContextModificationRequest{
		GNBCUUEF1APID: int64(ue.GnbCuUeF1apId),
		GNBDUUEF1APID: int64(ue.DuUeId),
		DRBsToBeSetupModList: []f1ies.DRBsToBeSetupModItem{{
			DRBID: int64(pduSession.DrbId),
			QoSInformation: f1ies.QoSInformation{
				Choice: f1ies.QoSInformationPresentEUTRANQoS,
				EUTRANQoS: &f1ies.EUTRANQoS{
					QCI: 9,
					AllocationAndRetentionPriority: f1ies.AllocationAndRetentionPriority{
						PriorityLevel: 1,
						PreEmptionCapability: f1ies.PreEmptionCapability{
							Value: f1ies.PreEmptionCapabilityShallnottriggerpreemption,
						},
						PreEmptionVulnerability: f1ies.PreEmptionVulnerability{
							Value: f1ies.PreEmptionVulnerabilityNotpreemptable,
						},
					},
				},
			},
			ULUPTNLInformationToBeSetupList: []f1ies.ULUPTNLInformationToBeSetupItem{{
				ULUPTNLInformation: f1ies.UPTransportLayerInformation{
					Choice: f1ies.UPTransportLayerInformationPresentGTPTunnel,
					GTPTunnel: &f1ies.GTPTunnel{
						TransportLayerAddress: aper.BitString{
							Bytes:   []byte{0xC0, 0xA8, 0x01, 0x64},
							NumBits: 32,
						},
						GTPTEID: []byte{
							byte(pduSession.UlTeid >> 24),
							byte(pduSession.UlTeid >> 16),
							byte(pduSession.UlTeid >> 8),
							byte(pduSession.UlTeid),
						},
					},
				},
			}},
			RLCMode: f1ies.RLCMode{
				Value: f1ies.RLCModeRlcam,
			},
		}},
	}

	f1apBytes, err := f1ap.F1apEncode(&msg)
	if err != nil {
		return fmt.Errorf("failed to encode F1AP UE Context Modification Request: %w", err)
	}

	err = duCtx.SendF1ap(f1apBytes)
	if err != nil {
		return fmt.Errorf("failed to send F1AP message to DU: %w", err)
	}

	cu.Info("F1AP UE Context Modification Request sent to DU %d", duCtx.DuId)
	return nil
}

func (cu *CuCpContext) handleF1UEContextModificationResponse(
	msg *f1ies.UEContextModificationResponse,
) error {
	cu.Info("Processing F1AP UE Context Modification Response")

	ue, err := cu.GetUEByF1Id(msg.GNBCUUEF1APID)
	if err != nil {
		return fmt.Errorf("UE not found for CU-UE-F1AP-ID %d: %v", msg.GNBCUUEF1APID, err)
	}

	if msg.DRBsSetupModList != nil {
		for _, drb := range msg.DRBsSetupModList {
			drbId := uint8(drb.DRBID)
			cu.Info("DRB ID=%d setup successful at DU", drbId)

			var pduSession *uecontext.PduSessionContext
			for _, ps := range ue.PduSessions {
				if ps.DrbId == drbId {
					pduSession = ps
					break
				}
			}

			if pduSession == nil {
				cu.Error("No PDU session found for DRB ID=%d", drbId)
				continue
			}

			cu.Info("TODO: Extract DL GTP Tunnel info from DRB setup response")
		}
	}

	cu.Info("TODO: E1AP Bearer Context Modification Request to CU-UP (out of scope)")

	err = cu.sendRRCReconfigurationForPduSession(ue)
	if err != nil {
		return fmt.Errorf("failed to send RRC Reconfiguration: %w", err)
	}

	return nil
}

func (cu *CuCpContext) sendRRCReconfigurationForPduSession(
	ue *uecontext.GNBUe,
) error {
	cu.Info("Building RRC Reconfiguration for PDU Session establishment")

	duCtx, err := cu.GetDUForUE(ue)
	if err != nil {
		return fmt.Errorf("DU not found for UE: %v", err)
	}

	var drbToAddModList []rrcies.DRB_ToAddMod
	var nasPduList []rrcies.DedicatedNAS_Message

	for _, pduSession := range ue.PduSessions {
		if pduSession.State == uecontext.PDU_SESSION_ESTABLISHING {
			drbToAddModList = append(drbToAddModList, rrcies.DRB_ToAddMod{
				Drb_Identity: rrcies.DRB_Identity{
					Value: uint64(pduSession.DrbId),
				},
				Pdcp_Config: &rrcies.PDCP_Config{
					Drb: &rrcies.PDCP_Config_drb{
						Pdcp_SN_SizeUL: &rrcies.PDCP_Config_drb_pdcp_SN_SizeUL{
							Value: rrcies.PDCP_Config_drb_pdcp_SN_SizeUL_Enum_len18bits,
						},
						Pdcp_SN_SizeDL: &rrcies.PDCP_Config_drb_pdcp_SN_SizeDL{
							Value: rrcies.PDCP_Config_drb_pdcp_SN_SizeDL_Enum_len18bits,
						},
						HeaderCompression: &rrcies.PDCP_Config_drb_headerCompression{
							Choice: rrcies.PDCP_Config_drb_headerCompression_Choice_NotUsed,
						},
					},
					T_Reordering: &rrcies.PDCP_Config_t_Reordering{
						Value: rrcies.PDCP_Config_t_Reordering_Enum_ms100,
					},
				},
			})

			if len(pduSession.NasPduSessionAccept) > 0 {
				nasPduList = append(nasPduList, rrcies.DedicatedNAS_Message{
					Value: pduSession.NasPduSessionAccept,
				})
			}
		}
	}

	masterCellGroupBytes, err := rrc.Encode(ue.MasterCellGroup)
	if err != nil {
		return fmt.Errorf("failed to encode MasterCellGroup: %w", err)
	}

	rrcmsg := rrcies.RRCReconfiguration{
		Rrc_TransactionIdentifier: rrcies.RRC_TransactionIdentifier{Value: 1},
		CriticalExtensions: rrcies.RRCReconfiguration_CriticalExtensions{
			Choice: rrcies.RRCReconfiguration_CriticalExtensions_Choice_RrcReconfiguration,
			RrcReconfiguration: &rrcies.RRCReconfiguration_IEs{
				RadioBearerConfig: &rrcies.RadioBearerConfig{
					Drb_ToAddModList: &rrcies.DRB_ToAddModList{
						Value: drbToAddModList,
					},
				},
				NonCriticalExtension: &rrcies.RRCReconfiguration_v1530_IEs{
					MasterCellGroup:          &masterCellGroupBytes,
					DedicatedNAS_MessageList: nasPduList,
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

	rrcBytes, err := rrc.Encode(&dlDcchMsg)
	if err != nil {
		return fmt.Errorf("failed to encode RRC Reconfiguration: %w", err)
	}

	f1rrcdl := f1ies.DLRRCMessageTransfer{
		GNBCUUEF1APID:      int64(ue.GnbCuUeF1apId),
		GNBDUUEF1APID:      int64(ue.DuUeId),
		SRBID:              1,
		RRCContainer:       rrcBytes,
		ExecuteDuplication: &f1ies.ExecuteDuplication{Value: 0},
	}

	f1apBytes, err := f1ap.F1apEncode(&f1rrcdl)
	if err != nil {
		return fmt.Errorf("failed to encode F1AP DL RRC Message Transfer: %w", err)
	}

	err = duCtx.SendF1ap(f1apBytes)
	if err != nil {
		return fmt.Errorf("failed to send RRC Reconfiguration to DU: %w", err)
	}

	cu.Info("RRC Reconfiguration sent to DU %d for PDU Session establishment", duCtx.DuId)
	return nil
}

func (cu *CuCpContext) sendPduSessionResourceSetupResponse(
	ue *uecontext.GNBUe,
) error {
	cu.Info("Building NGAP PDU Session Resource Setup Response")

	var setupList []ies.PDUSessionResourceSetupItemSURes

	for _, pduSession := range ue.PduSessions {
		if pduSession.State == uecontext.PDU_SESSION_ACTIVE {
			transferBytes := []byte{}

			setupItem := ies.PDUSessionResourceSetupItemSURes{
				PDUSessionID:                            int64(pduSession.PduSessionId),
				PDUSessionResourceSetupResponseTransfer: transferBytes,
			}

			setupList = append(setupList, setupItem)
		}
	}

	msg := ies.PDUSessionResourceSetupResponse{
		AMFUENGAPID:                      ue.AmfUeNgapId,
		RANUENGAPID:                      ue.RanUeNgapId,
		PDUSessionResourceSetupListSURes: setupList,
	}

	ngapBytes, err := ngap.NgapEncode(&msg)
	if err != nil {
		return fmt.Errorf("failed to encode PDU Session Resource Setup Response: %w", err)
	}

	amf, err := cu.GetAMFById(ue.AmfId)
	if err != nil {
		return fmt.Errorf("AMF not found: %v", err)
	}

	err = amf.SendNgap(ngapBytes)
	if err != nil {
		return fmt.Errorf("failed to send NGAP message: %w", err)
	}

	cu.Info("NGAP PDU Session Resource Setup Response sent to AMF")
	return nil
}
