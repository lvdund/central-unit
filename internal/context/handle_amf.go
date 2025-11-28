package context

import (
	"central-unit/internal/context/amfcontext"
	"central-unit/internal/context/du"
	"central-unit/internal/context/uecontext"
	"central-unit/internal/transport"
	"central-unit/pkg/model"
	"fmt"

	"github.com/alitto/pond/v2"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/ies"
)

func (cu *CuCpContext) newAmf(amfs model.AMF) *amfcontext.GNBAmf {
	amf := &amfcontext.GNBAmf{
		AmfId:   cu.getRanAmfId(),
		AmfIp:   amfs.Ip,
		AmfPort: amfs.Port,
		State:   amfcontext.AMF_INACTIVE,
	}
	cu.AmfPool.Store(amf.AmfId, amf)

	return amf
}

func (cu *CuCpContext) initAmfConn(amf *amfcontext.GNBAmf) error {
	// check AMF IP and AMF port.
	remote := fmt.Sprintf("%s:%d", amf.AmfIp, amf.AmfPort)
	local := fmt.Sprintf("%s:%d", cu.ControlInfo.gnbIp, cu.ControlInfo.gnbPort)

	conn := transport.NewSctpConn(cu.ControlInfo.gnbId, local, remote, cu.Ctx)
	if err := conn.Connect(); err != nil {
		cu.Fatal("Create SCTP connection err:", err)
	}
	amf.Tnla.SctpConn = conn
	cu.ControlInfo.n2 = conn

	p_amf := pond.NewPool(100)

	go func() {
		for rawMsg := range amf.Tnla.SctpConn.Read() {
			p_amf.Submit(func() { cu.dispatch(amf, rawMsg) })
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
		cu.Error("Error decoding NGAP message in %s GNB: %v", cu.ControlInfo.gnbId, err)
	}

	switch ngapMsg.Present {
	case ies.NgapPduSuccessfulOutcome:
		switch ngapMsg.Message.ProcedureCode.Value {

		case ies.ProcedureCode_NGSetup:
			cu.Info("Receive NG Setup Response")
			innerMsg := ngapMsg.Message.Msg.(*ies.NGSetupResponse)
			cu.handlerNgSetupResponse(amf, innerMsg)

		case ies.ProcedureCode_DownlinkNASTransport:
			cu.Info("Receive Downlink NAS Transport")
			innerMsg := ngapMsg.Message.Msg.(*ies.DownlinkNASTransport)
			cu.handleNgDownlinkNasTransport(amf, innerMsg)

		case ies.ProcedureCode_InitialContextSetup:
			cu.Info("Receive Initial Context Setup Request")
			innerMsg := ngapMsg.Message.Msg.(*ies.InitialContextSetupRequest)
			cu.handlerInitialContextSetupRequest(amf, innerMsg)

		default:
			cu.Warn("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
		}

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

	cu.IsReady <- true
}

func (cu *CuCpContext) handleNgDownlinkNasTransport(amf *amfcontext.GNBAmf, msg *ies.DownlinkNASTransport) {
	ranUeId := msg.RANUENGAPID
	// amfUeId := msg.AMFUENGAPID
	// messageNas := msg.NASPDU

	du_target, _ := cu.DuPool.Load(0) //WARN: now only work with 1 du
	if du_target == nil {
		cu.Error(
			"Cannot send DownlinkNASTransport message to UE: unknow UE RANUEID:%d",
			ranUeId)
		return
	}

	// send NAS message to UE.
	// cu.sendNasToUe(messageNas)
	du_target.(*du.GNBDU).SendF1ap(msg.NASPDU)
}

func (cu *CuCpContext) handlerInitialContextSetupRequest(amf *amfcontext.GNBAmf, msg *ies.InitialContextSetupRequest) {
	// ranUeId := msg.RANUENGAPID
	// amfUeId := msg.AMFUENGAPID
	// messageNas := msg.NASPDU
	var allowednssai []model.Snssai
	var mobilityRestrict = "not informed"
	var maskedImeisv string
	var ueSecurityCapabilities ies.UESecurityCapabilities
	// var pDUSessionResourceSetupListCxtReq []ies.PDUSessionResourceSetupItemCxtReq

	// var securityKey []byte
	//TODO: using for create new security context between GNB and UE.
	// securityKey = msg.SecurityKey.Value.Bytes

	allowednssai = make([]model.Snssai, len(msg.AllowedNSSAI))

	// list S-NSSAI(Single - Network Slice Selection Assistance Information).
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

	getdata, _ := cu.RrcUePool.Load(0)
	ue := getdata.(*uecontext.GNBUe)
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
