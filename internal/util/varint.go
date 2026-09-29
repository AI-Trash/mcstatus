package util

import (
	"errors"
	"io"
)

// ErrVarIntTooBig is returned when a VarInt exceeds 5 bytes.
var ErrVarIntTooBig = errors.New("VarInt is too big")

// WriteVarInt writes a 32-bit integer as a Minecraft protocol VarInt to w.
func WriteVarInt(val int32, w io.Writer) error {
	for {
		if (val & ^0x7F) == 0 {
			_, err := w.Write([]byte{byte(val)})
			return err
		}
		if _, err := w.Write([]byte{byte((val & 0x7F) | 0x80)}); err != nil {
			return err
		}
		val = int32(uint32(val) >> 7)
	}
}

// ReadVarInt reads a 32-bit Minecraft protocol VarInt from r.
func ReadVarInt(r io.Reader) (int32, error) {
	var num int32
	var count uint
	b := make([]byte, 1)

	for {
		if _, err := io.ReadFull(r, b); err != nil {
			return 0, err
		}

		num |= int32(b[0]&0x7F) << (7 * count)
		count++

		if count > 5 {
			return 0, ErrVarIntTooBig
		}

		if (b[0] & 0x80) == 0 {
			break
		}
	}

	return num, nil
}
