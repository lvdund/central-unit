# how to use RRC:

## RRC Library for Open RAN UE↔DU/CU Messaging

This Go module packages ASN.1-derived RRC message definitions and helpers so a User Equipment (UE) implementation can construct, encode, and decode signaling exchanged with the Distributed Unit (DU) and interpreted at the Centralized Unit (CU) inside an Open RAN architecture. It keeps the UE-side control-plane logic aligned with the DU/CU expectations by reusing the same IE layouts and UPER encoding rules.

## Usage

1. **Install the module**

   ```bash
   go get github.com/lvdund/rrc
   ```

2. **Import the packages you need** (core helpers plus the generated IE sets):

   ```go
   import (
       "github.com/lvdund/rrc"
       "github.com/lvdund/rrc/ies"
       "github.com/lvdund/asn1go/uper"
   )
   ```

3. **Wrap your IE in the `rrc.RRCMessage` interface** so it can be serialized and parsed:

   ```go
   type RRCSetupComplete struct {
       ies.RRCSetupComplete
   }

   func (m *RRCSetupComplete) Encode(w *uper.UperWriter) error {
       return m.RRCSetupComplete.Encode(w)
   }

   func (m *RRCSetupComplete) Decode(r *uper.UperReader) error {
       return m.RRCSetupComplete.Decode(r)
   }
   ```

4. **Call the helpers from UE signaling handlers when talking to the DU/CU:**

   ```go
   payload, err := rrc.Encode(&RRCSetupComplete{ /* set IE fields */ })
   if err != nil {
       // handle encoding error
   }

   msg, err := rrc.Decode(payload)
   if err != nil {
       // handle decoding error
   }
   ```

   `rrc.Encode` and `rrc.Decode` manage the UPER writer/reader plumbing so that UE-originated RRC PDUs remain spec-compliant when exchanged with the DU and later consumed by the CU.




# How to use F1ap:

```go
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

	pdu, err, diagnostics := F1apDecode(buf.Bytes())
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
```



