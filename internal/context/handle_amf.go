package context

import (
	"central-unit/internal/common/logger"
	"central-unit/internal/context/amfcontext"
	"central-unit/internal/transport"
	"central-unit/pkg/model"
	"fmt"

	f1ap "github.com/JocelynWS/f1-gen"
	f1ies "github.com/JocelynWS/f1-gen/ies"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/rrc"
	rrcies "github.com/lvdund/rrc/ies"
)

func (cu *CuCpContext) newAmf(amfs model.AMF) *amfcontext.GNBAmf {
	amf := &amfcontext.GNBAmf{
		AmfId:   cu.getRanAmfId(),
		AmfIp:   amfs.Ip,
		AmfPort: amfs.Port,
		State:   amfcontext.AMF_INACTIVE,
		Logger:  logger.InitLogger("", map[string]string{"mod": "amf"}),
	}
	cu.Info("==== Store AMF %d ====", amf.AmfId)
	// cu.AmfPool.Store(amf.AmfId, amf)
	cu.AMF = amf

	return amf
}

func (cu *CuCpContext) initAmfConn(amf *amfcontext.GNBAmf) error {
	// check AMF IP and AMF port.
	remote := fmt.Sprintf("%s:%d", amf.AmfIp, amf.AmfPort)
	local := fmt.Sprintf("%s:%d", cu.ControlInfo.ng_gnbIp, cu.ControlInfo.ng_gnbPort)

	conn := transport.NewSctpConn(cu.ControlInfo.ng_gnbId, local, remote, cu.Ctx)
	if err := conn.Connect(); err != nil {
		cu.Fatal("Create SCTP connection err: %s", err.Error())
	}
	amf.Tnla.SctpConn = conn
	cu.ControlInfo.n2 = conn

	// listen NGAP messages from AMF.
	go func() {
		for rawMsg := range cu.ControlInfo.n2.Read() {
			go cu.dispatch(amf, rawMsg)
		}
	}()
	return nil
}

func (cu *CuCpContext) dispatch(amf *amfcontext.GNBAmf, rawMsg []byte) {
	if len(rawMsg) == 0 {
		cu.Error("NGAP message is empty")
		return
	}

	ngapMsg, err, _ := ngap.NgapDecode(rawMsg)
	if err != nil {
		cu.Error("Error decoding NGAP message in %s GNB: %v", cu.ControlInfo.ng_gnbId, err)
	}
	cu.Info("Receive NGAP message", ngapMsg.Present, ngapMsg.Message.ProcedureCode.Value)

	switch ngapMsg.Present {

	case ies.NgapPduInitiatingMessage:
		switch ngapMsg.Message.ProcedureCode.Value {
		case ies.ProcedureCode_DownlinkNASTransport:
			cu.Info("Receive Downlink NAS Transport")
			innerMsg := ngapMsg.Message.Msg.(*ies.DownlinkNASTransport)
			cu.handleNgDownlinkNasTransport(amf, innerMsg)
		case ies.ProcedureCode_InitialContextSetup:
			cu.Info("Receive Initial Context Setup Request")
			innerMsg := ngapMsg.Message.Msg.(*ies.InitialContextSetupRequest)
			cu.handlerInitialContextSetupRequest(amf, innerMsg)
		default:
			cu.Warn("Received unknown NgapPduInitiatingMessage ProcedureCode 0x%x", ngapMsg.Message.ProcedureCode.Value)
		}
	case ies.NgapPduSuccessfulOutcome:
		switch ngapMsg.Message.ProcedureCode.Value {
		case ies.ProcedureCode_NGSetup:
			cu.Info("Receive NG Setup Response")
			innerMsg := ngapMsg.Message.Msg.(*ies.NGSetupResponse)
			cu.handlerNgSetupResponse(amf, innerMsg)
		default:
			cu.Warn("Received unknown NgapPduSuccessfulOutcome ProcedureCode 0x%x", ngapMsg.Message.ProcedureCode.Value)
		}
	default:
		cu.Warn("Received unknown NGAP message present 0x%x", ngapMsg.Present)
	}

}

func (cu *CuCpContext) handlerNgSetupResponse(amf *amfcontext.GNBAmf, msg *ies.NGSetupResponse) {
	cu.Info("Receive NGSetupResponse")
	var plmn string

	amfName := msg.AMFName
	amf.Name = string(amfName)

	amf.RelativeAmfCapacity = msg.RelativeAMFCapacity

	for _, items := range msg.PLMNSupportList {

		plmn = fmt.Sprintf("%x", items.PLMNIdentity)
		amf.AddedPlmn(plmn)

		for _, slice := range items.SliceSupportList {
			amf.AddedSlice(fmt.Sprintf("%x", slice.SNSSAI.SST), fmt.Sprintf("%x", slice.SNSSAI.SD))
		}
	}

	amf.State = amfcontext.AMF_ACTIVE
	cu.Info("AMF Name: %s - state: Active - capacity: %d", amf.Name, amf.RelativeAmfCapacity)
	for i := range amf.LenPlmn {
		mcc, mnc := amf.GetPlmnSupport(i)
		cu.Info("\tPLMNs Identities Supported by AMF -- mcc:%s mnc:%s", mcc, mnc)
	}
	for i := range amf.LenSlice {
		sst, sd := amf.GetSliceSupport(i)
		cu.Info("\tList of AMF slices Supported by AMF -- sst:%s sd:%s", sst, sd)
	}

	cu.IsReadyNgap <- true
}

func (cu *CuCpContext) handleNgDownlinkNasTransport(amf *amfcontext.GNBAmf, msg *ies.DownlinkNASTransport) {
	duCtx := cu.DU
	ue := cu.UE

	ue.AmfUeNgapId = msg.AMFUENGAPID

	rrcmsg := rrcies.DL_DCCH_Message{
		Message: rrcies.DL_DCCH_MessageType{
			Choice: rrcies.DL_DCCH_MessageType_Choice_C1,
			C1: &rrcies.DL_DCCH_MessageType_C1{
				Choice: rrcies.DL_DCCH_MessageType_C1_Choice_DlInformationTransfer,
				DlInformationTransfer: &rrcies.DLInformationTransfer{
					Rrc_TransactionIdentifier: rrcies.RRC_TransactionIdentifier{Value: 0},
					CriticalExtensions: rrcies.DLInformationTransfer_CriticalExtensions{
						Choice: rrcies.DLInformationTransfer_CriticalExtensions_Choice_DlInformationTransfer,
						DlInformationTransfer: &rrcies.DLInformationTransfer_IEs{
							DedicatedNAS_Message: &rrcies.DedicatedNAS_Message{
								Value: msg.NASPDU,
							},
						},
					},
				},
			},
		},
	}

	buf, err := rrc.Encode(&rrcmsg)
	if err != nil {
		cu.Error("Error encoding DL RRC Message Transfer: %v", err)
		return
	}

	f1rrcdl := f1ies.DLRRCMessageTransfer{
		GNBCUUEF1APID: int64(amf.AmfId),
		GNBDUUEF1APID: int64(duCtx.DuId),
		SRBID:         0,
		RRCContainer:  buf,
		ExecuteDuplication: &f1ies.ExecuteDuplication{
			Value: 0,
		},
		RedirectedRRCmessage: []byte{0},
	}

	f1apBytes, err := f1ap.F1apEncode(&f1rrcdl)
	if err != nil {
		cu.Error("Error encoding DL RRC Message Transfer: %v", err)
		return
	}
	err = duCtx.SendF1ap(f1apBytes)
	if err != nil {
		cu.Error("Error sending Downlink NAS Transport to DU: %v", err)
	}
	cu.Info("Send DL RRC Message Transfer to .DU %d", duCtx.DuId)
}

func (cu *CuCpContext) handlerInitialContextSetupRequest(amf *amfcontext.GNBAmf, msg *ies.InitialContextSetupRequest) {

	var allowednssai []model.Snssai
	var mobilityRestrict = "not informed"
	var maskedImeisv string
	var ueSecurityCapabilities ies.UESecurityCapabilities

	allowednssai = make([]model.Snssai, len(msg.AllowedNSSAI))

	for i, items := range msg.AllowedNSSAI {
		allowednssai[i] = model.Snssai{}

		if items.SNSSAI.SST != nil {
			allowednssai[i].Sst = fmt.Sprintf("%x", items.SNSSAI.SST)
		} else {
			allowednssai[i].Sst = "not informed"
		}

		if items.SNSSAI.SD != nil {
			allowednssai[i].Sd = fmt.Sprintf("%x", items.SNSSAI.SD)
		} else {
			allowednssai[i].Sd = "not informed"
		}
	}

	// that field is not mandatory.
	if msg.MobilityRestrictionList == nil {
		cu.Info("Mobility Restriction is missing")
		mobilityRestrict = "not informed"
	} else {
		mobilityRestrict = fmt.Sprintf("%x", msg.MobilityRestrictionList.ServingPLMN)
	}

	// that field is not mandatory.
	// TODO using for mapping UE context
	if msg.MaskedIMEISV == nil {
		cu.Info("Masked IMEISV is missing")
		maskedImeisv = "not informed"
	} else {
		maskedImeisv = fmt.Sprintf("%x", msg.MaskedIMEISV)
	}

	// TODO using for create new security context between UE and cu.
	// TODO algorithms for create new security context between UE and cu.
	ueSecurityCapabilities = msg.UESecurityCapabilities

	// if msg.PDUSessionResourceSetupListCxtReq == nil {
	// 	cu.Warn("PDUSessionResourceSetupListCxtReq is missing")
	// }
	// pDUSessionResourceSetupListCxtReq = msg.PDUSessionResourceSetupListCxtReq

	ue := cu.UE
	ue.CreateUeContext(mobilityRestrict, maskedImeisv, allowednssai, &ueSecurityCapabilities)

	// show UE context.
	cu.Info(" Context was created with successful")
	cu.Info(" RAN ID %d", ue.RanUeNgapId)
	cu.Info(" AMF ID %d", ue.AmfUeNgapId)
	cu.Info(" Mobility Restrict --Plmn-- Mcc:%s Mnc:%s", ue.MobilityInfo.Mcc, ue.MobilityInfo.Mnc)
	cu.Info(" Masked Imeisv: %s", ue.MaskedIMEISV)
	cu.Info(" lowed Nssai (Sst-Sd): %v", allowednssai)

	ue.RegistrationAccept = msg.NASPDU
	// getDUdata, _ := cu.DuPool.Load(0)
	// duCtx := getDUdata.(*du.GNBDU)
	// if msg.NASPDU != nil {
	// 	duCtx.SendF1ap(msg.NASPDU)
	// }
}
