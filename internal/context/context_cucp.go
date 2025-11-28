package context

import (
	"central-unit/internal/common/logger"
	"central-unit/internal/transport"
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/utils"
)

type CuCpContext struct {
	ControlInfo ControlInfo // gnb control plane information

	MsinPool  sync.Map // map[string]*GNBUe, Msin as key
	RrcUePool sync.Map // map[in64]*GNBUe, RrcUeId as key
	PrUePool  sync.Map // map[in64]*GNBUe, PrUeId as key
	TeidPool  sync.Map // map[uint32]*GNBUe, downlinkTeid as key

	AmfPool sync.Map // map[int64]*GNBAmf, AmfId as key
	DuPool  sync.Map // map[int64]*DU, DuId as key

	SliceInfo      Slice
	IdUeGenerator  int64  // ran UE id.
	IdAmfGenerator int64  // ran amf id
	TeidGenerator  uint32 // ran UE downlink Teid
	UeIpGenerator  uint8  // ran ue ip.

	// OAI
	IdRrcUeGenerator int64

	// check
	*logger.Logger
	Ctx     context.Context
	Mu      sync.Mutex
	IsReady chan bool
	Close   chan struct{}
}

type Slice struct {
	sd  string
	sst string
}

type ControlInfo struct {
	mcc     string
	mnc     string
	tac     string
	gnbId   string
	gnbIp   string
	gnbPort int
	// inboundChannel chan rlink.Message
	rlinkPool sync.Map
	n2        *transport.SctpConn
}

func (cu *CuCpContext) GetMccAndMncInOctets() []byte {
	var res string

	// reverse mcc and mnc
	mcc := reverse(cu.ControlInfo.mcc)
	mnc := reverse(cu.ControlInfo.mnc)

	if len(mnc) == 2 {
		res = fmt.Sprintf("%c%cf%c%c%c", mcc[1], mcc[2], mcc[0], mnc[0], mnc[1])
	} else {
		res = fmt.Sprintf("%c%c%c%c%c%c", mcc[1], mcc[2], mnc[2], mcc[0], mnc[0], mnc[1])
	}

	resu, _ := hex.DecodeString(res)
	return resu
}

func reverse(s string) string {
	// reverse string.
	var aux string
	for _, valor := range s {
		aux = string(valor) + aux
	}
	return aux
}

func (cu *CuCpContext) getGnbIdInBytes() []byte {
	// changed for bytes.
	resu, err := hex.DecodeString(cu.ControlInfo.gnbId)
	if err != nil {
		cu.Error("can not get gnbid in byte")
	}
	return resu
}

func (cu *CuCpContext) getSliceInBytes() ([]byte, []byte) {
	sstBytes, err := hex.DecodeString(cu.SliceInfo.sst)
	if err != nil {
		cu.Error("can not get Slice-sst in byte")
	}

	if cu.SliceInfo.sd != "" {
		sdBytes, err := hex.DecodeString(cu.SliceInfo.sd)
		if err != nil {
			cu.Error("can not get Slice-sd in byte")
		}
		return sstBytes, sdBytes
	}
	return sstBytes, nil
}

func (cu *CuCpContext) getTacInBytes() []byte {
	// changed for bytes.
	resu, err := hex.DecodeString(cu.ControlInfo.tac)
	if err != nil {
		cu.Error("can not get Tac in byte")
	}
	return resu
}

func (cu *CuCpContext) getRanAmfId() int64 {

	// TODO implement mutex

	id := cu.IdAmfGenerator

	// increment Amf Id
	cu.IdAmfGenerator++

	return id
}

// SetControlInfoFromConfig sets the control information from config values
func (cu *CuCpContext) SetControlInfoFromConfig(mcc, mnc, gnbIp, gnbId, tac string, gnbPort int) {
	cu.ControlInfo.mcc = mcc
	cu.ControlInfo.mnc = mnc
	cu.ControlInfo.gnbIp = gnbIp
	cu.ControlInfo.gnbPort = gnbPort
	cu.ControlInfo.gnbId = gnbId
	cu.ControlInfo.tac = tac
}

// SetSliceInfoFromConfig sets the slice information from config values
func (cu *CuCpContext) SetSliceInfoFromConfig(sst, sd string) {
	cu.SliceInfo.sst = sst
	cu.SliceInfo.sd = sd
}

func (cu *CuCpContext) GetPLMNIdentity() []byte {
	return utils.PlmnIdToNgap(utils.PlmnId{Mcc: cu.ControlInfo.mcc, Mnc: cu.ControlInfo.mnc})
}

func (cu *CuCpContext) GetNRCellIdentity() aper.BitString {
	nci := cu.getGnbIdInBytes()
	var slice = make([]byte, 2)

	return aper.BitString{
		Bytes:   append(nci, slice...),
		NumBits: 36,
	}
}

func (cu *CuCpContext) IsReadyCheck() bool {
	t := time.NewTicker(3 * time.Second)
	select {
	case <-cu.IsReady:
		return true
	case <-t.C:
		return false
	}
}

func (cu *CuCpContext) Terminate() {
	cu.Close <- struct{}{}

	// close(cu.ControlInfo.InboundChannel)
	cu.Info("NAS channel Terminated")

	n2 := cu.ControlInfo.n2
	if n2 != nil {
		cu.Info("N2/TNLA Terminated")
		n2.Close()
	}

	cu.Info("CU-CP Terminated")
}
