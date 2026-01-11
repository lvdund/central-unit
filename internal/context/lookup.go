package context

import (
	"fmt"

	"central-unit/internal/context/amfcontext"
	"central-unit/internal/context/du"
	"central-unit/internal/context/uecontext"
	"github.com/ishidawataru/sctp"
)

func (cu *CuCpContext) GetDUByConn(conn *sctp.SCTPConn) (*du.GNBDU, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}
	duIdVal, ok := cu.F1ConnMap.Load(conn)
	if !ok {
		return nil, fmt.Errorf("no DU registered for connection %v", conn.RemoteAddr())
	}
	return cu.GetDUById(duIdVal.(int64))
}

func (cu *CuCpContext) GetDUById(duId int64) (*du.GNBDU, error) {
	duVal, ok := cu.DuPool.Load(duId)
	if !ok {
		return nil, fmt.Errorf("DU %d not found in pool", duId)
	}
	return duVal.(*du.GNBDU), nil
}

func (cu *CuCpContext) GetDUForUE(ue *uecontext.GNBUe) (*du.GNBDU, error) {
	return cu.GetDUById(int64(ue.DuId))
}

func (cu *CuCpContext) GetUEByRrcId(rrcUeId int64) (*uecontext.GNBUe, error) {
	ueVal, ok := cu.RrcUePool.Load(rrcUeId)
	if !ok {
		return nil, fmt.Errorf("UE with RrcId %d not found", rrcUeId)
	}
	return ueVal.(*uecontext.GNBUe), nil
}

func (cu *CuCpContext) GetUEByF1Id(cuUeF1apId int64) (*uecontext.GNBUe, error) {
	ueVal, ok := cu.F1UePool.Load(cuUeF1apId)
	if !ok {
		return nil, fmt.Errorf("UE with F1AP-ID %d not found", cuUeF1apId)
	}
	return ueVal.(*uecontext.GNBUe), nil
}

func (cu *CuCpContext) GetUEByNgapId(ranUeNgapId int64) (*uecontext.GNBUe, error) {
	ueVal, ok := cu.NgapUePool.Load(ranUeNgapId)
	if !ok {
		return nil, fmt.Errorf("UE with RAN-UE-NGAP-ID %d not found", ranUeNgapId)
	}
	return ueVal.(*uecontext.GNBUe), nil
}

func (cu *CuCpContext) GetPrimaryAMF() (*amfcontext.GNBAmf, error) {
	var primaryAmf *amfcontext.GNBAmf
	cu.AmfPool.Range(func(key, value any) bool {
		if amf, ok := value.(*amfcontext.GNBAmf); ok {
			if amf.State == amfcontext.AMF_ACTIVE {
				primaryAmf = amf
				return false
			}
		}
		return true
	})
	if primaryAmf == nil {
		return nil, fmt.Errorf("no active AMF available")
	}
	return primaryAmf, nil
}

func (cu *CuCpContext) GetAMFById(amfId int64) (*amfcontext.GNBAmf, error) {
	amfVal, ok := cu.AmfPool.Load(amfId)
	if !ok {
		return nil, fmt.Errorf("AMF %d not found in pool", amfId)
	}
	return amfVal.(*amfcontext.GNBAmf), nil
}

func (cu *CuCpContext) RemoveUE(ue *uecontext.GNBUe) {
	cu.RrcUePool.Delete(int64(ue.RrcUeId))
	cu.NgapUePool.Delete(ue.RanUeNgapId)
	cu.F1UePool.Delete(int64(ue.GnbCuUeF1apId))

	cu.Info("Removed UE: RrcId=%d, NgapId=%d, F1Id=%d from all pools",
		ue.RrcUeId, ue.RanUeNgapId, ue.GnbCuUeF1apId)
}

func (cu *CuCpContext) RemoveDU(duCtx *du.GNBDU) {
	cu.DuPool.Delete(duCtx.DuId)
	if duCtx.SctpConn != nil {
		cu.F1ConnMap.Delete(duCtx.SctpConn)
	}
	cu.Info("Removed DU: %d from all pools", duCtx.DuId)
}

func (cu *CuCpContext) GetConnectedDUCount() int {
	count := 0
	cu.DuPool.Range(func(_, value any) bool {
		if d, ok := value.(*du.GNBDU); ok && d.IsActive() {
			count++
		}
		return true
	})
	return count
}

func (cu *CuCpContext) GetConnectedUECount() int {
	count := 0
	cu.RrcUePool.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}
