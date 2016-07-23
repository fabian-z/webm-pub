package varint

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "testing"
)

func TestNumPrecedingZeros(t *T) {
	m := map[byte]byte{
		0x80: 0,
		0x40: 1,
		0x20: 2,
		0x10: 3,
		0x08: 4,
		0x04: 5,
		0x02: 6,
		0x01: 7,
		0x00: 8,
	}

	assert := assert.New(t)
	for in, out := range m {
		assert.Equal(out, numPrecedingZeros(in), "input: %x", in)
	}
}

var m = map[VarInt]int64{
	VarInt(0x81):               0x01,
	VarInt(0xC1):               0x41,
	VarInt(0x4121):             0x0121,
	VarInt(0x53ac):             0x13ac,
	VarInt(0x204121):           0x4121,
	VarInt(0x234121):           0x034121,
	VarInt(0x0321123456789a):   0x0121123456789a,
	VarInt(0x014121123456789a): 0x4121123456789a,
}

func TestVarInt(t *T) {
	for in, out := range m {

		// First we need a buffer filled with the encoded varint
		buf := bytes.NewBuffer(make([]byte, 0, 8))
		_, err := in.WriteTo(buf)
		require.Nil(t, err, "input: 0x%x", in)

		size, err := in.Size()
		require.Nil(t, err, "input: 0x%x", in)
		assert.Equal(t, buf.Len(), size, "input: 0x%x", in)

		// Read the varint back off that buffer, to make sure it will be read
		// properly
		v, err := Read(buf)
		require.Nil(t, err, "input: 0x%x", in)

		// Make sure it's uint64 form is correct
		i, err := v.Uint64()
		require.Nil(t, err, "input: 0x%x", in)
		assert.Equal(t, out, i, "input: 0x%x", in)
	}
}
