package util_test

import (
	"bytes"
	"testing"

	"mcstatus/internal/util"
)

func TestVarIntRoundTrip(t *testing.T) {
	numbers := []int32{0, 1, 2, 127, 128, 255, 256, 754, 765, 776, 2147483647, -1, -2147483648}

	for _, n := range numbers {
		var buf bytes.Buffer
		if err := util.WriteVarInt(n, &buf); err != nil {
			t.Fatalf("WriteVarInt(%d) failed: %v", n, err)
		}

		got, err := util.ReadVarInt(&buf)
		if err != nil {
			t.Fatalf("ReadVarInt for %d failed: %v", n, err)
		}

		if got != n {
			t.Errorf("Roundtrip mismatch: wrote %d, read %d", n, got)
		}
	}
}

func TestVarIntProtocol776Encoding(t *testing.T) {
	// Protocol 776 in binary: 0b00000011 00001000
	// 776 & 0x7F = 0x08 -> 0x88 (first byte with continue bit)
	// 776 >> 7 = 6 -> 0x06 (second byte)
	// Expected bytes: [0x88, 0x06]
	var buf bytes.Buffer
	if err := util.WriteVarInt(776, &buf); err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x88, 0x06}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Fatalf("Protocol 776 encoding want %x, got %x", expected, buf.Bytes())
	}
}
