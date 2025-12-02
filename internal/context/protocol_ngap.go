package context

import (
	"central-unit/internal/context/amfcontext"
	"central-unit/internal/context/uecontext"

	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
)

func (cu *CuCpContext) SendNasPdu(
	nasPdu []byte,
	ue *uecontext.GNBUe,
	amf *amfcontext.GNBAmf,
) {
	var buf []byte
	var err error
	switch ue.State {
	case uecontext.UE_INITIALIZED:
		buf, err = cu.ngInitialUEMessage(nasPdu, ue)
		ue.State = uecontext.UE_ONGOING
		if err != nil {
			cu.Error("Encode NG Initial UE Message: %s", err.Error())
			return
		}
	case uecontext.UE_ONGOING, uecontext.UE_READY:
		buf, err = cu.ngUplinkNasTransport(nasPdu, ue)
		if err != nil {
			cu.Error("Encode NG Uplink NAS Transport: %s", err.Error())
			return
		}
	}

	cu.Info("Sending NGAP to AMF")
	// _ = buf
	amf.SendNgap(buf)
	if err != nil {
		cu.Error("Error sending Nas message in NGAP: %s", err.Error())
	}
}

func (cu *CuCpContext) SendNgSetupRequest(amf *amfcontext.GNBAmf) {
	cu.Info("Initiating NG Setup Request")

	msg := ies.NGSetupRequest{}

	msg.GlobalRANNodeID = ies.GlobalRANNodeID{
		Choice: ies.GlobalRANNodeIDPresentGlobalgnbId,
		GlobalGNBID: &ies.GlobalGNBID{
			PLMNIdentity: cu.GetMccAndMncInOctets(),
			GNBID: ies.GNBID{
				Choice: ies.GNBIDPresentGnbId,
				GNBID: &aper.BitString{
					Bytes:   cu.getGnbIdInBytes(),
					NumBits: 24,
				},
			},
		},
	}

	msg.RANNodeName = []byte("cu-cp")

	sst, sd := cu.getSliceInBytes()
	msg.SupportedTAList = []ies.SupportedTAItem{
		{
			TAC: cu.getTacInBytes(),
			BroadcastPLMNList: []ies.BroadcastPLMNItem{
				{
					PLMNIdentity: cu.GetMccAndMncInOctets(),
					TAISliceSupportList: []ies.SliceSupportItem{
						{SNSSAI: ies.SNSSAI{SST: sst, SD: sd}},
					},
				},
			},
		},
	}

	msg.DefaultPagingDRX = ies.PagingDRX{Value: ies.PagingDRXV128}

	ngapPdu, err := ngap.NgapEncode(&msg)
	if err != nil {
		cu.Error("Error sending NG Setup Request: ", err)
	}

	cu.Info("Sending NG Setup Request to AMF %s", amf.Name)
	amf.SendNgap(ngapPdu)
	if err != nil {
		cu.Error("Error sending NG Setup Request: ", err)
	}
}

func (cu *CuCpContext) ngInitialUEMessage(
	nasPdu []byte,
	ue *uecontext.GNBUe,
) ([]byte, error) {
	cu.Info("Create InitialUeMessage NGAP")

	tac := cu.getTacInBytes()
	plmn := cu.GetPLMNIdentity()
	cellid := cu.GetNRCellIdentity()

	msg := ies.InitialUEMessage{
		RANUENGAPID: ue.RanUeNgapId,
		NASPDU:      nasPdu,
		UserLocationInformation: ies.UserLocationInformation{
			Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
			UserLocationInformationNR: &ies.UserLocationInformationNR{
				NRCGI: ies.NRCGI{
					NRCellIdentity: cellid,
					PLMNIdentity:   plmn,
				},
				TAI: ies.TAI{
					PLMNIdentity: plmn,
					TAC:          tac,
				},
			},
		},
		RRCEstablishmentCause: ies.RRCEstablishmentCause{
			Value: ies.RRCEstablishmentCauseMosignalling,
		},
		FiveGSTMSI: ue.Tmsi5gs,
	}

	return ngap.NgapEncode(&msg)
}

func (cu *CuCpContext) ngUplinkNasTransport(
	nasPdu []byte,
	ue *uecontext.GNBUe,
) ([]byte, error) {
	msg := ies.UplinkNASTransport{
		AMFUENGAPID: ue.AmfUeNgapId,
		RANUENGAPID: ue.RanUeNgapId,
		NASPDU:      nasPdu,
		UserLocationInformation: ies.UserLocationInformation{
			Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
			UserLocationInformationNR: &ies.UserLocationInformationNR{
				NRCGI: ies.NRCGI{
					NRCellIdentity: cu.GetNRCellIdentity(),
					PLMNIdentity:   cu.GetPLMNIdentity(),
				},
				TAI: ies.TAI{
					PLMNIdentity: cu.GetPLMNIdentity(),
					TAC:          cu.getTacInBytes(),
				},
			},
		},
	}
	return ngap.NgapEncode(&msg)
}
