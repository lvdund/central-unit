package context

import (
	"central-unit/internal/context/amfcontext"
	"central-unit/internal/context/uecontext"

	"github.com/lvdund/asn1go/uper"
)

// Based on rrc_gNB_create_ue_context from OAI
func (cu *CuCpContext) createUE(
	duid int64,
	crnti int64,
	ueIdentity uper.BitString,
	duUeId int64,
) *uecontext.GNBUe {
	cu.Mu.Lock()
	rrcUeid := cu.IdRrcUeGenerator
	cu.IdRrcUeGenerator++
	cu.Mu.Unlock()

	amf, _ := cu.AmfPool.Load(0)

	ue := uecontext.GNBUe{
		DuId:               uint64(duid),
		Rnti:               crnti,
		Random_ue_identity: ueIdentity.Bytes,
		AmfUeNgapId:        0, //WARN
		RanUeNgapId:        0, //WARN
		RrcUeId:            uint64(rrcUeid),
		DuUeId:             uint64(duUeId),
		AmfId:              amf.(*amfcontext.GNBAmf).AmfId,
	}

	cu.RrcUePool.Store(rrcUeid, ue)

	return &ue
}
