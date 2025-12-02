package amfcontext

import (
	"central-unit/internal/common/logger"
	"central-unit/internal/transport"
	"fmt"

	"github.com/lvdund/ngap/aper"
)

// AMF main states in the GNB Context.
const (
	AMF_INACTIVE   string = "AMF_INACTIVE"
	AMF_ACTIVE     string = "AMF_ACTIVE"
	AMF_OVERLOADED string = "AMF_OVERLOAD"
)

type GNBAmf struct {
	*logger.Logger
	AmfIp               string         // AMF ip
	AmfPort             int            // AMF port
	AmfId               int64          // AMF id
	Tnla                TNLAssociation // AMF sctp associations
	RelativeAmfCapacity int64          // AMF capacity
	State               string
	Name                string // amf name.
	RegionId            aper.BitString
	SetId               aper.BitString
	Pointer             aper.BitString
	Plmns               *PlmnSupported
	Slices              *SliceSupported
	LenSlice            int
	LenPlmn             int
	BackupAMF           string
	// TODO implement the other fields of the AMF Context
}

type TNLAssociation struct {
	SctpConn         *transport.SctpConn
	TnlaWeightFactor int64
	Usage            aper.Enumerated
	Streams          uint16
}

type SliceSupported struct {
	Sst    string
	Sd     string
	Status string
	Next   *SliceSupported
}

type PlmnSupported struct {
	Mcc  string
	Mnc  string
	Next *PlmnSupported
}

func (amf *GNBAmf) GetSliceSupport(index int) (string, string) {

	mov := amf.Slices
	for range index {
		mov = mov.Next
	}

	return mov.Sst, mov.Sd
}

func (amf *GNBAmf) GetPlmnSupport(index int) (string, string) {

	mov := amf.Plmns
	for range index {
		mov = mov.Next
	}

	return mov.Mcc, mov.Mnc
}

func ConvertMccMnc(plmn string) (mcc string, mnc string) {
	if plmn[2] == 'f' {
		mcc = fmt.Sprintf("%c%c%c", plmn[1], plmn[0], plmn[3])
		mnc = fmt.Sprintf("%c%c", plmn[5], plmn[4])
	} else {
		mcc = fmt.Sprintf("%c%c%c", plmn[1], plmn[0], plmn[3])
		mnc = fmt.Sprintf("%c%c%c", plmn[2], plmn[5], plmn[4])
	}

	return mcc, mnc
}

func (amf *GNBAmf) AddedPlmn(plmn string) {

	if amf.LenPlmn == 0 {
		newElem := &PlmnSupported{}

		// newElem.info = plmn
		newElem.Next = nil
		newElem.Mcc, newElem.Mnc = ConvertMccMnc(plmn)
		// update list
		amf.Plmns = newElem
		amf.LenPlmn++
		return
	}

	mov := amf.Plmns
	for range amf.LenPlmn {

		// end of the list
		if mov.Next == nil {

			newElem := &PlmnSupported{}
			newElem.Mcc, newElem.Mnc = ConvertMccMnc(plmn)
			newElem.Next = nil

			mov.Next = newElem

		} else {
			mov = mov.Next
		}
	}

	amf.LenPlmn++
}

func (amf *GNBAmf) AddedSlice(sst string, sd string) {

	if amf.LenSlice == 0 {
		newElem := &SliceSupported{}
		newElem.Sst = sst
		newElem.Sd = sd
		newElem.Next = nil

		// update list
		amf.Slices = newElem
		amf.LenSlice++
		return
	}

	mov := amf.Slices
	for range amf.LenSlice {

		// end of the list
		if mov.Next == nil {

			newElem := &SliceSupported{}
			newElem.Sst = sst
			newElem.Sd = sd
			newElem.Next = nil

			mov.Next = newElem

		} else {
			mov = mov.Next
		}
	}
	amf.LenSlice++
}

func (amf *GNBAmf) SetRegionId(regionId []byte) {
	amf.RegionId = aper.BitString{
		Bytes:   regionId,
		NumBits: uint64(len(regionId) * 8),
	}
}

func (amf *GNBAmf) SetSetId(setId []byte) {
	amf.SetId = aper.BitString{
		Bytes:   setId,
		NumBits: uint64(len(setId) * 8),
	}
}

func (amf *GNBAmf) SetPointer(pointer []byte) {
	amf.Pointer = aper.BitString{
		Bytes:   pointer,
		NumBits: uint64(len(pointer) * 8),
	}
}

func (amf *GNBAmf) SendNgap(pdu []byte) error {
	err := amf.Tnla.SctpConn.Send(pdu)
	if err != nil {
		amf.Error("Error sending NGAP message: %v", err)
		return err
	}
	amf.Info("Sent NGAP message to AMF %d", amf.AmfId)
	return nil
}
