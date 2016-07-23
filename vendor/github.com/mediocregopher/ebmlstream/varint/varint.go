// Implementation of the variable-sized integer for the ebml document type.
// varints are integers which are encoded in such a way as to take up only as
// many bytes as needed (smaller integers take up fewer bytes). This is done
// through a utf-8 style method where the number of preceding zero bits in the
// first byte of the integer indicates how many further bytes need to be read to
// encompass the full integer
package varint

import (
	"bytes"
	"errors"
	"io"
)

var (
	IntegerTooBig = errors.New("integer too big")
	InvalidVarInt = errors.New("invalid var int")
)

const (
	// The largest unsigned integer which can be represented with the VarInt
	// data type
	MaxEncodable = uint64(0xffffffffffffff)

	// The minimum and maximum raw varints which can exist. The represent
	// MaxEncodable and 0, respectively
	maxRaw = VarInt((MaxEncodable + 1) | MaxEncodable)
	minRaw = VarInt(0x80)
)

type VarInt uint64

func numPrecedingZeros(b byte) byte {
	for i := byte(0); i < 8; i++ {
		if b&0x80 > 0 {
			return i
		}
		b <<= 1
	}
	return 8
}

func readByte(r io.Reader) (byte, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r, b)
	return b[0], err
}

// Reads an encoded variable integer from the given reader, reading only as many
// bytes as necessary. This will keep the VarInt exactly as it was read, even if
// the form it was read in was not as compact as possible. Use Normalize() to
// compact and existing VarInt
func Read(r io.Reader) (VarInt, error) {
	b, err := readByte(r)
	if err != nil {
		return 0, err
	}

	rem := numPrecedingZeros(b)
	ret := uint64(b)
	for ; rem > 0; rem-- {
		b, err = readByte(r)
		if err != nil {
			return 0, err
		}
		ret = (ret << 8) | uint64(b)
	}

	return VarInt(ret), nil
}

// Same as Read, but reads from an existing byte slice (without modifying the
// slice) The slice of bytes must have enough bytes to encompass the full
// varint, but having more bytes than necessary is ok
func Parse(b []byte) (VarInt, error) {
	buf := bytes.NewBuffer(b)
	return Read(buf)
}

// Encodes the given integer into the smallest possible VarInt
func Encode(i uint64) (VarInt, error) {
	if i > MaxEncodable {
		return 0, IntegerTooBig
	}

	// Count the number of bits actually being used by the number
	bits := byte(0)
	for ic := i; ; {
		ic >>= 1
		bits++
		if ic == 0 {
			break
		}
	}

	// Find the number of bytes the raw number uses. We round up using the cool
	// addition hack
	bytes := (bits + 7) / 8

	// If we have enough bits available at the front of the raw number to simply
	// drop the size bits there we don't need to add another byte to the front
	// of our encoded number
	bitsAvailable := (bytes * 8) - bits
	bitsNeeded := bytes
	if bitsNeeded > bitsAvailable {
		bytes++
	}

	one := 0x80 >> (bytes - 1)
	shifted := uint64(one) << ((bytes - 1) * 8)
	return VarInt(i | shifted), nil
}

// Returns the unencoded form of the VarInt
func (v VarInt) Uint64() (uint64, error) {
	if v > maxRaw || v < minRaw {
		return 0, InvalidVarInt
	}
	v64 := uint64(v)
	ret := v64
	for mask := ^uint64(0); ; mask >>= 1 {
		ret &= mask
		if ret != v64 {
			return uint64(ret), nil
		}
	}
}

// Returns the number of bytes the encoded form of this varint would take up if
// written out
func (v VarInt) Size() (int, error) {
	if v > maxRaw || v < minRaw {
		return 0, InvalidVarInt
	}
	mask := ^VarInt(0)
	for i := byte(1); ; i++ {
		mask <<= 8
		if v & mask == 0 {
			return int(i), nil
		}
	}
}

// Returns a VarInt of equivalent value to this one but in its most compact
// form.
func (v VarInt) Normalize() (VarInt, error) {
	i, err := v.Uint64()
	if err != nil {
		return 0, err
	}
	return Encode(i)
}

// Writes the VarInt in its encoded form to the given io.Writer, implementing
// the io.WriterTo interface
func (v VarInt) WriteTo(w io.Writer) (int, error) {
	if v > maxRaw || v < minRaw {
		return 0, InvalidVarInt
	}

	out := make([]byte, 0, 8)
	bytes := byte(1)
	for thresh := VarInt(0xff); ; thresh = (thresh << 8) | thresh {
		if v <= thresh {
			break
		}
		bytes++
	}

	for i := bytes - 1; i >= 0 && i < 255; i-- {
		b := byte((v >> (i * 8)) & 0xff)
		out = append(out, b)
	}
	return w.Write(out)
}
