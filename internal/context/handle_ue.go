package context

import (
	"central-unit/internal/context/uecontext"

	"github.com/lvdund/asn1go/aper"
)

// Based on rrc_gNB_create_ue_context from OAI
func (cu *CuCpContext) createUE(
	duid int64,
	crnti int64,
	ueIdentity aper.BitString,
	duUeId int64,
) *uecontext.GNBUe {
	rrcUeid := cu.IdRrcUeGenerator
	cu.IdRrcUeGenerator++

	// amf, ok := cu.AmfPool.Load(0)
	// if !ok {
	// 	cu.Error("AMF id 0 not found")
	// 	return nil
	// }
	amf := cu.AMF

	ue := uecontext.GNBUe{
		DuId:               uint64(duid),
		Rnti:               crnti,
		Random_ue_identity: ueIdentity.Bytes,
		AmfUeNgapId:        1, //WARN
		RanUeNgapId:        1, //WARN
		RrcUeId:            uint64(rrcUeid),
		DuUeId:             uint64(duUeId),
		AmfId:              amf.AmfId,
	}

	cu.Info("===== Store UE %d =====", rrcUeid)
	cu.UE = &ue

	return &ue
}
