package edtd

import (
	"container/list"
	"fmt"
	"io"

	"github.com/mediocregopher/ebmlstream"
)

// Parsers are generated from an Edtd using NewParser. They return sequential
// Elem structs which have their data already read in (meaning you DON'T have to
// call a data method before calling Next() again, as in the root ebmlstream
// package). See the example package for more on how to use the Parser
type Parser struct {
	edtd     *Edtd
	lastElem *ebmlstream.Elem
	buffer   *list.List
}

// Represents a single ebml element. It contains the base ebmlstream.Elem this
// is based on (with the data for that element having already been read into
// it), as well as some extra information from the edtd
type Elem struct {
	ebmlstream.Elem
	Type
	Name  string

	// The heirarchical level of the edtd this element appears on. Starts at 0
	// and goes up from there
	Level uint64
}

// Returns a new parser for the edtd which will read from the io.Reader and
// return Elems
func (e *Edtd) NewParser(r io.Reader) *Parser {
	return &Parser{
		edtd:     e,
		lastElem: ebmlstream.RootElem(r),
		buffer:   list.New(),
	}
}

// Returns the next ebml element in the stream. It is NOT necessary to call a
// data method on the Elem before calling Next() again (as it is in the base
// ebmlstream package)
func (p *Parser) Next() (*Elem, error) {
	if f := p.buffer.Front(); f != nil {
		return p.buffer.Remove(f).(*Elem), nil
	}

	e, err := p.lastElem.Next()
	if err != nil {
		return nil, err
	}

	etpl, ok := p.edtd.elements[elementID(e.Id)]
	if !ok {
		return nil, fmt.Errorf("unknown id: %x", e.Id)
	}

	switch etpl.typ {
	case Int:
		_, err = e.Int()
	case Uint:
		_, err = e.Uint()
	case Float:
		_, err = e.Float()
	case Date:
		_, err = e.Date()
	case String:
		_, err = e.Str()
	case Binary:
		_, err = e.Bytes()
	}
	if err != nil {
		return nil, err
	}

	p.lastElem = e

	return &Elem{
		Elem:  *e,
		Type:  etpl.typ,
		Name:  etpl.name,
		Level: etpl.level,
	}, nil
}
