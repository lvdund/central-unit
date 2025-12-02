package utils

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
)

func Build5GSTMSI(part2, part1 aper.BitString) []byte {
	// Convert both to big.Int
	v2 := new(big.Int).SetBytes(part2.Bytes)
	v1 := new(big.Int).SetBytes(part1.Bytes)

	// Shift part2 by 39 bits
	v2.Lsh(v2, 39)

	// OR
	res := new(big.Int).Or(v2, v1)

	// Get bytes
	b := res.Bytes()

	// Pad to 6 bytes (48 bits) big-endian
	const outLen = 6
	if len(b) > outLen {
		// Truncate from left (should not happen with valid input)
		b = b[len(b)-outLen:]
	} else if len(b) < outLen {
		tmp := make([]byte, outLen)
		copy(tmp[outLen-len(b):], b)
		b = tmp
	}
	return b
}

// Decode5GSTMSI decodes a 6-byte (48-bit) 5G-S-TMSI into its three components.
// Input: tmsi - 6-byte big-endian representation of 5G-S-TMSI
// Returns:
//
//	amfSetID: 2-byte big-endian (10-bit value, zero-extended)
//	amfPointer: 1-byte (6-bit value)
//	fivegTMSI: 4-byte big-endian (32-bit TMSI)
func Decode5GSTMSI(tmsi []byte) (*ies.FiveGSTMSI, error) {
	if len(tmsi) != 6 {
		return nil, fmt.Errorf("5G-S-TMSI (%v) must be 6 bytes", tmsi)
	}

	// Reconstruct 48-bit integer (as uint64)
	var val uint64
	for i := 0; i < 6; i++ {
		val = (val << 8) | uint64(tmsi[i])
	}

	amfSetIDVal := uint16(val >> 38)       // 10 bits
	amfPtrVal := uint8((val >> 32) & 0x3F) // 6 bits
	fivegTMSIVal := uint32(val)            // 32 bits

	amfSetID := make([]byte, 2)
	binary.BigEndian.PutUint16(amfSetID, amfSetIDVal)

	amfPointer := []byte{amfPtrVal}

	fivegTMSI := make([]byte, 4)
	binary.BigEndian.PutUint32(fivegTMSI, fivegTMSIVal)

	return &ies.FiveGSTMSI{
		AMFSetID:   aper.BitString{Bytes: amfSetID, NumBits: 10},
		AMFPointer: aper.BitString{Bytes: amfPointer, NumBits: 9},
		FiveGTMSI:  fivegTMSI,
	}, nil
}

func BitStringToUint64(asn *aper.BitString) uint64 {
	var result uint64
	bitsUnused := (len(asn.Bytes) * 8) - int(asn.NumBits)

	// DevCheck equivalent - panic if size constraints violated
	if len(asn.Bytes) == 0 || len(asn.Bytes) > 8 {
		panic(fmt.Sprintf("invalid BitString size: %d (must be 1-8)", len(asn.Bytes)))
	}

	shift := ((len(asn.Bytes) - 1) * 8) - bitsUnused

	// Process all bytes except the last one
	for index := 0; index < len(asn.Bytes)-1; index++ {
		result |= uint64(asn.Bytes[index]) << shift
		shift -= 8
	}

	// Process the last byte, shifting right by unused bits
	result |= uint64(asn.Bytes[len(asn.Bytes)-1]) >> bitsUnused

	return result
}

// extractRandomValue extracts random value from RRC Setup Request UE identity
func extractRandomValue(randomValueBytes []byte) uint64 {
	if len(randomValueBytes) == 0 {
		return 0
	}

	// Random value is typically 39 or 48 bits
	// For 39-bit: 5 bytes, but only 39 bits are used
	// For 48-bit: 6 bytes

	var value uint64
	if len(randomValueBytes) >= 5 {
		// Extract up to 5 bytes (40 bits max)
		for i := 0; i < len(randomValueBytes) && i < 5; i++ {
			value = (value << 8) | uint64(randomValueBytes[i])
		}
		// Right-shift if 39-bit (remove unused bits)
		if len(randomValueBytes) == 5 {
			value = value >> 1 // Remove the last bit if 39-bit
		}
	} else {
		// Extract available bytes
		for i := 0; i < len(randomValueBytes); i++ {
			value = (value << 8) | uint64(randomValueBytes[i])
		}
	}

	return value
}

// extractSTMSI extracts 5G-S-TMSI Part1 from bit string
func extractSTMSI(stmsiBytes []byte, numBits uint64) uint64 {
	if len(stmsiBytes) == 0 {
		return 0
	}

	var value uint64
	bitsToRead := int(numBits)
	if bitsToRead > 64 {
		bitsToRead = 64
	}

	for i := 0; i < len(stmsiBytes) && bitsToRead > 0; i++ {
		bits := 8
		if bitsToRead < 8 {
			bits = bitsToRead
		}
		value = (value << uint(bits)) | uint64(stmsiBytes[i]>>(8-bits))
		bitsToRead -= bits
	}

	return value
}

// extractUint64FromBytes extracts uint64 from byte array (big-endian)
func extractUint64FromBytes(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}

	// Pad to 8 bytes if needed
	padded := make([]byte, 8)
	copy(padded[8-len(data):], data)

	return binary.BigEndian.Uint64(padded)
}
