package context

import (
	"central-unit/internal/context/uecontext"

	"github.com/lvdund/asn1go/aper"
)

func (cu *CuCpContext) createUE(
	duid int64,
	crnti int64,
	ueIdentity aper.BitString,
	duUeId int64,
) *uecontext.GNBUe {
	rrcUeId := cu.getNextRrcUeId()
	ranUeNgapId := cu.getNextRanUeNgapId()
	gnbCuUeF1apId := cu.getNextGnbCuUeF1apId()

	amf, err := cu.GetPrimaryAMF()
	if err != nil {
		cu.Error("No AMF available for UE creation: %v", err)
		return nil
	}

	ue := &uecontext.GNBUe{
		DuId:               uint64(duid),
		Rnti:               crnti,
		Random_ue_identity: ueIdentity.Bytes,
		AmfUeNgapId:        0,
		RanUeNgapId:        ranUeNgapId,
		RrcUeId:            uint64(rrcUeId),
		DuUeId:             uint64(duUeId),
		GnbCuUeF1apId:      uint64(gnbCuUeF1apId),
		AmfId:              amf.AmfId,
		State:              uecontext.UE_INITIALIZED,
	}

	cu.RrcUePool.Store(rrcUeId, ue)
	cu.NgapUePool.Store(ranUeNgapId, ue)
	cu.F1UePool.Store(gnbCuUeF1apId, ue)

	cu.Info("Created UE: RrcId=%d, RanNgapId=%d, CuF1apId=%d, DuId=%d",
		rrcUeId, ranUeNgapId, gnbCuUeF1apId, duid)

	return ue
}
