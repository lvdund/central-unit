package du

import (
	"bytes"
	"fmt"
	"testing"

	f1ap "github.com/JocelynWS/f1-gen"
	"github.com/JocelynWS/f1-gen/ies"
	"github.com/lvdund/ngap/aper"
)

func TestF1SetupRequestMandatory(t *testing.T) {
	transactionID := int64(2)
	var bufTr bytes.Buffer
	w := aper.NewWriter(&bufTr)
	if err := w.WriteInteger(transactionID, &aper.Constraint{Lb: 0, Ub: 255}, false); err != nil {
		t.Fatalf("Encode TransactionID err: %v", err)
	}
	w.Close()

	fmt.Printf("TransactionID: %d (0x%x)\n", transactionID, transactionID)

	r := aper.NewReader(&bufTr)
	val, err := r.ReadInteger(&aper.Constraint{Lb: 0, Ub: 255}, false)
	if err != nil {
		t.Fatalf("Decode TransactionID err: %v", err)
	}
	if val != transactionID {
		t.Errorf("TransactionID mismatch: got %v, want %v", val, transactionID)
	}
	fmt.Printf("TransactionID Decoded: %d (0x%x)\n", val, val)
	fmt.Println("---")

	gnbduID := int64(1)
	var bufGNB bytes.Buffer
	w = aper.NewWriter(&bufGNB)
	if err := w.WriteInteger(gnbduID, &aper.Constraint{Lb: 0, Ub: 68719476735}, false); err != nil {
		t.Fatalf("Encode GNBDUID err: %v", err)
	}
	w.Close()

	fmt.Printf("GNBDUID: %d (0x%x)\n", gnbduID, gnbduID)

	r = aper.NewReader(&bufGNB)
	val, err = r.ReadInteger(&aper.Constraint{Lb: 0, Ub: 68719476735}, false)
	if err != nil {
		t.Fatalf("Decode GNBDUID err: %v", err)
	}
	if val != gnbduID {
		t.Errorf("GNBDUID mismatch: got %v, want %v", val, gnbduID)
	}
	fmt.Printf("GNBDUID Decoded: %d (0x%x)\n", val, val)
	fmt.Println("---")

	gnbduName := []byte("OAI DU")
	fmt.Printf("GNBDUName: %s (% x)\n", string(gnbduName), gnbduName)
	fmt.Println("---")

	plmn := []byte{0x21, 0x23, 0xF1}
	fmt.Printf("PLMN: % x (MCC=12, MNC=123)\n", plmn)
	fmt.Println("---")

	tac := []byte{0x00, 0x00, 0x01}
	fmt.Printf("Tracking Area Code: % x\n", tac)
	fmt.Println("---")

	sst1 := []byte{0x01}
	sst2 := []byte{0x02}
	sst3 := []byte{0x03}
	fmt.Printf("SST[0] (eMBB): % x\n", sst1)
	fmt.Printf("SST[1] (URLLC): % x\n", sst2)
	fmt.Printf("SST[2] (MIoT): % x\n", sst3)
	fmt.Println("---")

	rrcBytes := []byte{0x0c, 0x22, 0x38}
	rrcBits := uint64(3)
	rrc := aper.BitString{Bytes: rrcBytes, NumBits: rrcBits}

	var bufRRC bytes.Buffer
	w = aper.NewWriter(&bufRRC)
	if err := w.WriteBitString(rrc.Bytes, uint(rrc.NumBits), &aper.Constraint{Lb: 3, Ub: 3}, false); err != nil {
		t.Fatalf("Encode RRCVersion err: %v", err)
	}
	w.Close()

	fmt.Printf("RRCVersion: % x (%d bits)\n", rrc.Bytes, rrcBits)

	r = aper.NewReader(&bufRRC)
	bytesOut, nbitsOut, err := r.ReadBitString(&aper.Constraint{Lb: 3, Ub: 3}, false)
	if err != nil {
		t.Fatalf("Decode RRCVersion err: %v", err)
	}
	if uint64(nbitsOut) != rrc.NumBits || !bytes.Equal(bytesOut, rrc.Bytes[:1]) {
		t.Errorf("RRCVersion mismatch: got %v %d bits, want %v %d bits",
			bytesOut, nbitsOut, rrc.Bytes[:1], rrc.NumBits)
	}
	fmt.Printf("RRCVersion Decoded: % x (%d bits)\n", bytesOut, nbitsOut)
	fmt.Println("---")
}

func Test_F1SetupRequest(t *testing.T) {
	msg := ies.F1SetupRequest{
		TransactionID: 2,
		GNBDUID:       1,
		GNBDUName:     []byte("OAI DU"),
		GNBDURRCVersion: ies.RRCVersion{
			LatestRRCVersion: aper.BitString{
				Bytes:   []byte{0x0c, 0x22, 0x38},
				NumBits: 3,
			},
		},
	}

	fmt.Println("=== F1SetupRequest Encode (Mandatory Fields Only) ===")
	fmt.Printf("TransactionID: %d (0x%x)\n", msg.TransactionID, msg.TransactionID)
	fmt.Printf("GNBDUID: %d (0x%x)\n", msg.GNBDUID, msg.GNBDUID)
	fmt.Printf("GNBDUName: %s (% x)\n", string(msg.GNBDUName), msg.GNBDUName)
	fmt.Printf("RRCVersion: % x (%d bits)\n", msg.GNBDURRCVersion.LatestRRCVersion.Bytes, msg.GNBDURRCVersion.LatestRRCVersion.NumBits)

	var buf bytes.Buffer
	err := msg.Encode(&buf)
	if err != nil {
		fmt.Println("Encode error:", err)
	} else {
		fmt.Printf("\nFull Encoded Message: % x\n", buf.Bytes())
		fmt.Printf("Full Encoded Message (length %d bytes)\n", len(buf.Bytes()))
	}
}

func Test_F1SetupRequest_Decode(t *testing.T) {
	msg := ies.F1SetupRequest{
		TransactionID: 2,
		GNBDUID:       1,
		GNBDUName:     []byte("OAI DU"),
		GNBDURRCVersion: ies.RRCVersion{
			LatestRRCVersion: aper.BitString{
				Bytes:   []byte{0x0c, 0x22, 0x38},
				NumBits: 3,
			},
		},
	}

	fmt.Println("=== F1SetupRequest Full Message Encode/Decode ===")
	fmt.Println("\n--- Original Message ---")
	fmt.Printf("TransactionID: %d (0x%x)\n", msg.TransactionID, msg.TransactionID)
	fmt.Printf("GNBDUID: %d (0x%x)\n", msg.GNBDUID, msg.GNBDUID)
	fmt.Printf("GNBDUName: %s (% x)\n", string(msg.GNBDUName), msg.GNBDUName)
	fmt.Printf("RRCVersion: % x (%d bits)\n", msg.GNBDURRCVersion.LatestRRCVersion.Bytes, msg.GNBDURRCVersion.LatestRRCVersion.NumBits)

	var buf bytes.Buffer
	err := msg.Encode(&buf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	fmt.Printf("\n--- Encoded Bytes ---\n")
	fmt.Printf("Full Message: % x\n", buf.Bytes())
	fmt.Printf("Length: %d bytes\n", len(buf.Bytes()))

	pdu, err, diagnostics := f1ap.F1apDecode(buf.Bytes())
	if err != nil {
		t.Fatalf("Decode error: %v, diagnostics: %v", err, diagnostics)
	}

	decoded := pdu.Message.Msg.(*ies.F1SetupRequest)

	fmt.Printf("\n--- Decoded Message ---\n")
	fmt.Printf("TransactionID: %d (0x%x)\n", decoded.TransactionID, decoded.TransactionID)
	fmt.Printf("GNBDUID: %d (0x%x)\n", decoded.GNBDUID, decoded.GNBDUID)
	fmt.Printf("GNBDUName: %s (% x)\n", string(decoded.GNBDUName), decoded.GNBDUName)
	fmt.Printf("RRCVersion: % x (%d bits)\n", decoded.GNBDURRCVersion.LatestRRCVersion.Bytes, decoded.GNBDURRCVersion.LatestRRCVersion.NumBits)
}
