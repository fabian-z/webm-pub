package ebmlstream

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	. "testing"
)

func sb(bs ...byte) string {
	b := make([]byte, 0, len(bs))
	b = append(b, bs...)
	return string(b)
}

func TestIntElem(t *T) {
	m := map[string]int64{
		sb(0x80, 0x80):                   0,
		sb(0x80, 0x81, 0x01):             0x01,
		sb(0x80, 0x82, 0x02, 0x01):       0x0201,
		sb(0x80, 0x83, 0x03, 0x02, 0x01): 0x030201,
	}
	assert := assert.New(t)
	for in, out := range m {
		b := []byte(in)
		e, err := RootElem(bytes.NewBuffer(b)).Next()
		assert.Nil(err, "input: %x", in)

		i, err := e.Int()
		assert.Nil(err, "input: %x", in)
		assert.Exactly(out, i, "input: %x", in)

		ui, err := e.Uint()
		assert.Nil(err, "input: %x", in)
		assert.Exactly(uint64(out), ui, "input: %x", in)

		wbuf := bytes.NewBuffer([]byte{})
		i, err = e.WriteTo(wbuf)
		assert.Nil(err, "input: %x", in)
		assert.Equal(len(in), i, "input: %x", in)
		assert.Exactly(in, wbuf.String(), "input: %x", in)
	}
}

func TestStringElem(t *T) {
	m := map[string]string{
		sb(0x80, 0x80):                      "",
		sb(0x80, 0x83, 'f', 'o', 'o'):       "foo",
		sb(0x80, 0x85, 'f', 'o', 'o', 0, 0): "foo",
	}
	assert := assert.New(t)
	for in, out := range m {
		b := []byte(in)
		e, err := RootElem(bytes.NewBuffer(b)).Next()
		assert.Nil(err, "input: %x", in)

		s, err := e.Str()
		assert.Nil(err, "input: %x", in)
		assert.Equal(out, s, "input: %x", in)

		wbuf := bytes.NewBuffer([]byte{})
		i, err := e.WriteTo(wbuf)
		assert.Nil(err, "input: %x", in)
		assert.Equal(len(in), i, "input: %x", in)
		assert.Exactly(in, wbuf.String(), "input: %x", in)
	}
}
