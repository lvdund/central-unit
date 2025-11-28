package context

import (
	"bytes"
	"central-unit/internal/common/utils"
	"central-unit/internal/context/amfcontext"
	"central-unit/internal/context/du"
	"central-unit/internal/context/uecontext"
	"fmt"

	f1ap "github.com/JocelynWS/f1-gen"
	"github.com/JocelynWS/f1-gen/ies"
	f1ies "github.com/JocelynWS/f1-gen/ies"
	"github.com/lvdund/asn1go/uper"
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
	var ue *uecontext.GNBUe
	var err error

	if rrcSetupRequest.Ue_Identity.Choice == rrcies.InitialUE_Identity_Choice_RandomValue {
		ue = cu.createUE(duCtx.DuId, f1apMsg.CRNTI, uper.BitString{}, f1apMsg.GNBDUUEF1APID)
	} else if rrcSetupRequest.Ue_Identity.Choice == rrcies.InitialUE_Identity_Choice_Ng_5G_S_TMSI_Part1 {
		ue = cu.createUE(duCtx.DuId, f1apMsg.CRNTI, rrcSetupRequest.Ue_Identity.Ng_5G_S_TMSI_Part1, f1apMsg.GNBDUUEF1APID)
		ue.Tmsi5gs_part1 = &rrcSetupRequest.Ue_Identity.Ng_5G_S_TMSI_Part1
	} else {
		ue = cu.createUE(duCtx.DuId, f1apMsg.CRNTI, uper.BitString{}, f1apMsg.GNBDUUEF1APID)
		//TODO: rrc setup reject
	}

	if f1apMsg.DUtoCURRCContainer == nil {
		//TODO: rrc setup reject
		return fmt.Errorf("DUtoCURRCContainer nil")
	}
	//TODO: decode DUtoCURRCContainer -> cellGroupConfig type
	rrc_temp, err := rrc.Decode(f1apMsg.DUtoCURRCContainer)
	if err != nil {
		return err
	}

	ue.EstablishmentCause = &rrcSetupRequest.EstablishmentCause
	ue.NrCellId = &f1apMsg.NRCGI.NRCellIdentity
	ue.MasterCellGroup = rrc_temp.(*rrcies.CellGroupConfig)

	// Send RRC Setup -> DU
	rrcmsg := rrcies.RRCSetup{
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
		RedirectedRRCmessage: []byte{0}, //FIX: Now lib F1AP is wrong in this field
	}

	// Encode F1AP message
	var buf bytes.Buffer
	if err := dlRrcMsg.Encode(&buf); err != nil {
		return fmt.Errorf("encode DL RRC Message Transfer: %w", err)
	}

	// Send via SCTP to DU
	return duCtx.SendF1ap(buf.Bytes())
}

func (cu *CuCpContext) handleRrcSetupComplete(
	ue *uecontext.GNBUe,
	msg *rrcies.RRCSetupComplete,
) error {
	var tmsi5gs []byte
	tmsi := msg.CriticalExtensions.RrcSetupComplete.Ng_5G_S_TMSI_Value
	if tmsi != nil {
		if tmsi.Choice ==
			rrcies.RRCSetupComplete_IEs_ng_5G_S_TMSI_Value_Choice_Ng_5G_S_TMSI_Part2 &&
			ue.Tmsi5gs_part1 != nil {
			tmsi5gs = utils.Build5GSTMSI(
				aper.BitString(tmsi.Ng_5G_S_TMSI_Part2),
				aper.BitString(*ue.Tmsi5gs_part1))
		} else if tmsi.Choice ==
			rrcies.RRCSetupComplete_IEs_ng_5G_S_TMSI_Value_Choice_Ng_5G_S_TMSI {
			tmsi5gs = tmsi.Ng_5G_S_TMSI.Value.Bytes
		}

		ue.Tmsi5gs = utils.Decode5GSTMSI(tmsi5gs)
		ue.Random_ue_identity = tmsi5gs
	}

	ue.State = uecontext.UE_INITIALIZED

	amf, ok := cu.AmfPool.Load(0) //WARN: now fix id amf = 0
	if !ok {
		return fmt.Errorf("cannot load amf")
	}
	cu.SendNasPdu(msg.CriticalExtensions.RrcSetupComplete.DedicatedNAS_Message.Value, ue, amf.(*amfcontext.GNBAmf))
	return nil
}

func (cu *CuCpContext) handleULInformationTransfer(
	ue *uecontext.GNBUe,
	ulInformationTransfer *rrcies.ULInformationTransfer,
) error {
	ue.State = uecontext.UE_ONGOING
	amf, ok := cu.AmfPool.Load(0) //WARN: now fix id amf = 0
	if !ok {
		return fmt.Errorf("cannot load amf")
	}
	cu.SendNasPdu(ulInformationTransfer.CriticalExtensions.UlInformationTransfer.DedicatedNAS_Message.Value, ue, amf.(*amfcontext.GNBAmf))
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
			NRCellIdentity: *ue.NrCellId,
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

	duCtx, _ := cu.DuPool.Load(ue.DuUeId)
	err = duCtx.(*du.GNBDU).SendF1ap(f1apBytes)
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

	amf, ok := cu.AmfPool.Load(0) //WARN: now fix id amf = 0
	if !ok {
		return fmt.Errorf("cannot load amf")
	}

	err = amf.(*amfcontext.GNBAmf).SendNgap(ngapBytes)
	if err != nil {
		return fmt.Errorf("failed to send NGAP Initial Context Setup Response: %w", err)
	}

	cu.Info("NGAP Initial Context Setup Response sent successfully")
	return nil
}
