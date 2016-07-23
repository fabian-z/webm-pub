package edtd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	. "testing"
)

// TODO this needs tests
func mustDefDataBytes(d interface{}) []byte {
	b, err := defDataBytes(d)
	if err != nil {
		panic(err)
	}
	return b
}

// This is what implicitHeader should parse to

func TestParseImplicitElements(t *T) {
	implicitM := elementMap{
		0xa45dfa3: {
			id:   0xa45dfa3,
			typ:  Container,
			name: "EBML",
			card: oneOrMore,
		},
		0x286: {
			id:    0x286,
			typ:   Uint,
			name:  "EBMLVersion",
			def:   mustDefDataBytes(uint64(1)),
			level: 1,
		},
		0x2f7: {
			id:    0x2f7,
			typ:   Uint,
			name:  "EBMLReadVersion",
			def:   mustDefDataBytes(uint64(1)),
			level: 1,
		},
		0x2f2: {
			id:    0x2f2,
			typ:   Uint,
			name:  "EBMLMaxIDLength",
			def:   mustDefDataBytes(uint64(4)),
			level: 1,
		},
		0x2f3: {
			id:    0x2f3,
			typ:   Uint,
			name:  "EBMLMaxSizeLength",
			def:   mustDefDataBytes(uint64(8)),
			level: 1,
		},
		0x282: {
			id:     0x282,
			typ:    String,
			name:   "DocType",
			ranges: &rangeParam{loweri: 32, upperi: 126},
			level:  1,
		},
		0x287: {
			id:    0x287,
			typ:   Uint,
			name:  "DocTypeVersion",
			def:   mustDefDataBytes(uint64(1)),
			level: 1,
		},
		0x285: {
			id:    0x285,
			typ:   Uint,
			name:  "DocTypeReadVersion",
			def:   mustDefDataBytes(uint64(1)),
			level: 1,
		},

		// CRC32
		0x43: {
			id:   0x43,
			typ:  Container,
			name: "CRC32",
			card: zeroOrMore,
		},
		0x2fe: {
			id:    0x2fe,
			typ:   Binary,
			name:  "CRC32Value",
			size:  4,
			level: 1,
		},

		// Void
		0x6c: {
			id:   0x6c,
			typ:  Binary,
			name: "Void",
			card: zeroOrMore,
		},
	}

	e, err := NewEdtd(bytes.NewBufferString(""))
	require.Nil(t, err)
	assert.Equal(t, implicitM, e.elements)
}

func TestParseTypes(t *T) {

	test := `
        define types {
            bool := uint [ range:0..1; ]
            ascii := string [ range:32..126; ]
        }

        define elements {
		    Foo := 53ab bool [ def:1; ]
			Bar := 53ac bool [ card:?; ]
		}
	`

	e, err := NewEdtd(bytes.NewBufferString(test))
	require.Nil(t, err)

	boolRange := &rangeParam{
		lowerui: 0,
		upperui: 1,
	}

	foo := &tplElement{
		id:     0x13ab,
		typ:    Uint,
		name:   "Foo",
		def:    mustDefDataBytes(uint64(1)),
		ranges: boolRange,
	}
	assert.Equal(t, foo, e.elements[0x13ab])

	bar := &tplElement{
		id:     0x13ac,
		typ:    Uint,
		name:   "Bar",
		card:   zeroOrOnce,
		ranges: boolRange,
	}
	assert.Equal(t, bar, e.elements[0x13ac])
}

func TestParseFloatRange(t *T) {
	test := `
        define elements {
		    Foo := 53ab float [ def:1; range:>0.0 ]
		}
	`

	e, err := NewEdtd(bytes.NewBufferString(test))
	require.Nil(t, err)

	foo := &tplElement{
		id:   0x13ab,
		typ:  Float,
		name: "Foo",
		def:  mustDefDataBytes(float64(1)),
		ranges: &rangeParam{
			lowerf:  0.0,
			upperf:  math.MaxFloat64,
			exLower: true,
			exUpper: true,
		},
	}

	assert.Equal(t, foo, e.elements[0x13ab])
}
