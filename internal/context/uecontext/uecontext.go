package uecontext

import (
	"central-unit/internal/common/logger"
	"central-unit/internal/transport"
	"central-unit/pkg/model"
	"fmt"
	"sync"

	"github.com/lvdund/asn1go/uper"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/ngap/utils"
	rrcies "github.com/lvdund/rrc/ies"
)

// UE main states in the GNB Context.
const (
	UE_INITIALIZED uint8 = iota
	UE_ONGOING
	UE_READY
	UE_DOWN
)

type GNBUe struct {
	RanUeNgapId int64 // Identifier for UE in GNB Context.
	AmfUeNgapId int64 // Identifier for UE in AMF Context.
	AmfId       int64 // Identifier for AMF in UE/GNB Context.
	State       uint8 // State of UE in NAS/GNB Context.

	SctpConnection *transport.SctpConn // Sctp ue vs amf.

	Auth   AuthContext
	SecCtx SecurityContext

	// check
	*logger.Logger
	Lock sync.Mutex

	// oai
	RrcUeId            uint64
	DuId               uint64
	DuUeId             uint64
	Tmsi5gs_part1      *uper.BitString
	Tmsi5gs            *ies.FiveGSTMSI
	Rnti               int64
	Random_ue_identity []byte
	NrCellId           *aper.BitString
	MasterCellGroup    *rrcies.CellGroupConfig
	EstablishmentCause *rrcies.EstablishmentCause

	RegistrationAccept []byte

	// stormsim: UE context
	MobilityInfo           utils.PlmnId
	MaskedIMEISV           string
	AllowedSnssai          []model.Snssai
	LenSlice               int
	UeSecurityCapabilities *ies.UESecurityCapabilities
	// PduSession             [16]*GnbPDUSession
}

func (ue *GNBUe) CreateUeContext(plmn string, imeisv string, allowednssai []model.Snssai, ueSecurityCapabilities *ies.UESecurityCapabilities) {
	if plmn != "not informed" {
		ue.MobilityInfo.Mcc, ue.MobilityInfo.Mnc = convertMccMnc(plmn)
	} else {
		ue.MobilityInfo.Mcc = plmn
		ue.MobilityInfo.Mnc = plmn
	}

	ue.MaskedIMEISV = imeisv
	ue.AllowedSnssai = allowednssai
	ue.UeSecurityCapabilities = ueSecurityCapabilities
}

func convertMccMnc(plmn string) (mcc string, mnc string) {
	if plmn[2] == 'f' {
		mcc = fmt.Sprintf("%c%c%c", plmn[1], plmn[0], plmn[3])
		mnc = fmt.Sprintf("%c%c", plmn[5], plmn[4])
	} else {
		mcc = fmt.Sprintf("%c%c%c", plmn[1], plmn[0], plmn[3])
		mnc = fmt.Sprintf("%c%c%c", plmn[2], plmn[5], plmn[4])
	}

	return mcc, mnc
}
