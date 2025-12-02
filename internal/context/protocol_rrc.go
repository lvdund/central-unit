package context

import (
	"central-unit/internal/common/utils"
	"central-unit/internal/context/du"
	"central-unit/internal/context/uecontext"
	"fmt"

	f1ap "github.com/JocelynWS/f1-gen"
	"github.com/JocelynWS/f1-gen/ies"
	f1ies "github.com/JocelynWS/f1-gen/ies"
	asn1aper "github.com/lvdund/asn1go/aper"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	ngapies "github.com/lvdund/ngap/ies"
	"github.com/lvdund/rrc"
	rrcies "github.com/lvdund/rrc/ies"
)

// Based on rrc_handle_RRCSetupRequest from OAI
func (cu *CuCpContext) handleRRCSetupRequest(
	duCtx *du.GNBDU,
	rrcSetupRequest *rrcies.RRCSetupRequest_IEs,
	f1apMsg *ies.InitialULRRCMessageTransfer,
) error {
	cu.Info("handle RRC Setup Request")
	var ue *uecontext.GNBUe
	var err error

	switch rrcSetupRequest.Ue_Identity.Choice {
	case rrcies.InitialUE_Identity_Choice_RandomValue:
		ue = cu.createUE(duCtx.DuId, f1apMsg.CRNTI, asn1aper.BitString{}, f1apMsg.GNBDUUEF1APID)
	case rrcies.InitialUE_Identity_Choice_Ng_5G_S_TMSI_Part1:
		ue = cu.createUE(duCtx.DuId, f1apMsg.CRNTI, rrcSetupRequest.Ue_Identity.Ng_5G_S_TMSI_Part1, f1apMsg.GNBDUUEF1APID)
		ue.Tmsi5gs_part1 = (*aper.BitString)(&rrcSetupRequest.Ue_Identity.Ng_5G_S_TMSI_Part1)
	default:
		ue = cu.createUE(duCtx.DuId, f1apMsg.CRNTI, asn1aper.BitString{}, f1apMsg.GNBDUUEF1APID)
		//TODO: rrc setup reject
		return fmt.Errorf("invalid UE identity choice")
	}

	if f1apMsg.DUtoCURRCContainer == nil {
		//TODO: rrc setup reject
		return fmt.Errorf("DUtoCURRCContainer nil")
	} else {
		rrc_temp := rrcies.CellGroupConfig{}
		err = rrc.Decode(f1apMsg.DUtoCURRCContainer, &rrc_temp)
		if err != nil {
			return err
		}
		ue.MasterCellGroup = &rrc_temp
	}
	ue.EstablishmentCause = &rrcSetupRequest.EstablishmentCause
	ue.NrCellId = &f1apMsg.NRCGI.NRCellIdentity

	// Send RRC Setup -> DU
	rrcmsg := rrcies.DL_CCCH_Message{
		Message: rrcies.DL_CCCH_MessageType{
			Choice: rrcies.DL_CCCH_MessageType_Choice_C1,
			C1: &rrcies.DL_CCCH_MessageType_C1{
				Choice: rrcies.DL_CCCH_MessageType_C1_Choice_RrcSetup,
				RrcSetup: &rrcies.RRCSetup{
					Rrc_TransactionIdentifier: rrcies.RRC_TransactionIdentifier{
						Value: 0,
					},
					CriticalExtensions: rrcies.RRCSetup_CriticalExtensions{
						Choice: rrcies.RRCSetup_CriticalExtensions_Choice_RrcSetup,
						RrcSetup: &rrcies.RRCSetup_IEs{
							RadioBearerConfig: rrcies.RadioBearerConfig{
								Srb_ToAddModList: &rrcies.SRB_ToAddModList{
									Value: []rrcies.SRB_ToAddMod{rrcies.SRB_ToAddMod{
										Srb_Identity: rrcies.SRB_Identity{
											Value: 1,
										},
									}},
								},
							},
							MasterCellGroup: f1apMsg.DUtoCURRCContainer,
						},
					},
				},
			},
		},
	}
	rrcSetupBytes, err := rrc.Encode(&rrcmsg)
	if err != nil {
		return fmt.Errorf("failed to generate RRC Setup message: %v", err)
	}

	// Create DL RRC Message Transfer message
	dlRrcMsg := f1ies.DLRRCMessageTransfer{
		GNBCUUEF1APID:        int64(ue.RrcUeId),
		GNBDUUEF1APID:        int64(ue.DuUeId),
		SRBID:                0, //= 0 (SRB0, used before SRB1 is established)
		RRCContainer:         rrcSetupBytes,
		ExecuteDuplication:   &f1ies.ExecuteDuplication{Value: 0},
		RedirectedRRCmessage: []byte{0}, //FIX: Now lib F1AP is wrong in this field
	}

	f1apBytes, err := f1ap.F1apEncode(&dlRrcMsg)
	if err != nil {
		cu.Error("failed to encode DL RRC Message Transfer: %v", err)
		return fmt.Errorf("failed to encode DL RRC Message Transfer: %v", err)
	}

	cu.Info("Send RrcSetup to DU %d", duCtx.DuId)

	// Send via SCTP to DU
	return duCtx.SendF1ap(f1apBytes)
}

func (cu *CuCpContext) handleRrcSetupComplete(
	ue *uecontext.GNBUe,
	msg *rrcies.RRCSetupComplete,
) error {
	var tmsi5gs []byte
	var err error
	tmsi := msg.CriticalExtensions.RrcSetupComplete.Ng_5G_S_TMSI_Value
	if tmsi != nil {
		if tmsi.Choice == rrcies.RRCSetupComplete_IEs_ng_5G_S_TMSI_Value_Choice_Ng_5G_S_TMSI_Part2 &&
			ue.Tmsi5gs_part1 != nil {
			tmsi5gs = utils.Build5GSTMSI(aper.BitString(tmsi.Ng_5G_S_TMSI_Part2), aper.BitString(*ue.Tmsi5gs_part1))
		} else if tmsi.Choice ==
			rrcies.RRCSetupComplete_IEs_ng_5G_S_TMSI_Value_Choice_Ng_5G_S_TMSI {
			tmsi5gs = tmsi.Ng_5G_S_TMSI.Value.Bytes
		}

		if len(tmsi5gs) > 0 {
			ue.Tmsi5gs, err = utils.Decode5GSTMSI(tmsi5gs)
			if err != nil {
				return fmt.Errorf("failed to decode 5G-S-TMSI: %w", err)
			}
			ue.Random_ue_identity = tmsi5gs
		}
	}

	ue.State = uecontext.UE_INITIALIZED

	cu.Info("Send NAS Registration Request to AMF")
	amf := cu.AMF
	cu.SendNasPdu(msg.CriticalExtensions.RrcSetupComplete.DedicatedNAS_Message.Value, ue, amf)
	return err
}

func (cu *CuCpContext) handleULInformationTransfer(
	ue *uecontext.GNBUe,
	ulInformationTransfer *rrcies.ULInformationTransfer,
) error {
	ue.State = uecontext.UE_ONGOING
	// amf, ok := cu.AmfPool.Load(0) //WARN: now fix id amf = 0
	// if !ok {
	// 	return fmt.Errorf("cannot load amf")
	// }
	amf := cu.AMF
	cu.SendNasPdu(ulInformationTransfer.CriticalExtensions.UlInformationTransfer.DedicatedNAS_Message.Value, ue, amf)
	return nil
}

func (cu *CuCpContext) handleRRCSecurityModeComplete(
	ue *uecontext.GNBUe,
	securityModeComplete *rrcies.SecurityModeComplete,
) error {

	duUeId := int64(ue.DuUeId)
	msg := f1ies.UEContextSetupRequest{
		GNBCUUEF1APID: int64(ue.RrcUeId),
		GNBDUUEF1APID: &duUeId,
		SpCellID: f1ies.NRCGI{
			PLMNIdentity:   cu.GetMccAndMncInOctets(),
			NRCellIdentity: aper.BitString(*ue.NrCellId),
		},
		ServCellIndex: 0,
		CUtoDURRCInformation: &f1ies.CUtoDURRCInformation{
			CGConfigInfo: []byte{0x00}, //FIX: this field is not mandatory
		},
		SRBsToBeSetupList: []f1ies.SRBsToBeSetupItem{{
			SRBID: 2, //SRB2
		}},
		DRBsToBeSetupList: []f1ies.DRBsToBeSetupItem{{
			DRBID: 0, //because no pdu session is established yet
		}},
		NRUESidelinkAggregateMaximumBitrate: &f1ies.NRUESidelinkAggregateMaximumBitrate{
			UENRSidelinkAggregateMaximumBitrate: 1000000000,
		},
		ConditionalInterDUMobilityInformation: &f1ies.ConditionalInterDUMobilityInformation{
			CHOTrigger: f1ies.CHOTriggerInterDU{
				Value: f1ies.CHOtriggerInterDUChoinitiation,
			},
		},
	}

	f1apBytes, err := f1ap.F1apEncode(&msg)
	if err != nil {
		return fmt.Errorf("failed to encode UE Context Setup Request: %w", err)
	}

	// duCtx, _ := cu.DuPool.Load(ue.DuUeId)
	duCtx := cu.DU
	err = duCtx.SendF1ap(f1apBytes)
	if err != nil {
		return fmt.Errorf("failed to send UE Context Setup Request: %w", err)
	}

	return nil
}

func (cu *CuCpContext) handleRRCReconfigurationComplete(
	ue *uecontext.GNBUe,
	rrcReconfigurationComplete *rrcies.RRCReconfigurationComplete,
) error {
	ue.State = uecontext.UE_READY

	msg := &ngapies.InitialContextSetupResponse{
		AMFUENGAPID: ue.AmfUeNgapId,
		RANUENGAPID: ue.RanUeNgapId,
	}

	ngapBytes, err := ngap.NgapEncode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode Initial Context Setup Response: %w", err)
	}

	// amf, ok := cu.AmfPool.Load(0) //WARN: now fix id amf = 0
	// if !ok {
	// 	return fmt.Errorf("cannot load amf")
	// }
	amf := cu.AMF
	err = amf.SendNgap(ngapBytes)
	if err != nil {
		return fmt.Errorf("failed to send NGAP Initial Context Setup Response: %w", err)
	}

	cu.Info("NGAP Initial Context Setup Response sent successfully")
	return nil
}
