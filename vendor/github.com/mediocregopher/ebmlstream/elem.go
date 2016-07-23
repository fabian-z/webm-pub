// A package for reading an ebmlstream and the data which can be retrieved from
// it.
//
// Example usage (ids 0x2 and 0x3 are children of id 0x1, and are expected to be
// embl strings):
//
//	var err error
//	e := ebmlstream.RootElem(r)
//	for {
//		e, err = e.Next()
//		if err != nil {
//			return err
//		}
//
//		if e.Id == 0x1 {
//			fmt.Printf("%x - container\n", e.Id)
//		} else {
//			s, _ := e.Str()
//			fmt.Printf("%x - %s\n", e.Id, s)
//		}
//	}
package ebmlstream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/mediocregopher/ebmlstream/varint"
)

// Represents a single EBML element. EBML elements have only three properties:
// a numeric id, a size (in bytes) and their actual data. The id and size can be
// retrieved as fields on this struct, and data can be retrieved using one of
// the methods (depending on the data type).
//
// When an Elem is retrieved (using Next()) and it is not a container element it
// MUST have one of the data methods called (e.g. Int(), Bytes(), etc...) before
// Next() is called again, as this is what causes the data to be actually read
// from the reader. Data methods can be called multiple times, and different
// ones can be called, but at least one MUST be called before Next(). If the
// element is a container element then ONLY Next() can be called on it (although
// it will still have Id and Size filled in).
type Elem struct {
	r   io.Reader
	buf *bufio.Reader
	data []byte

	Id   varint.VarInt
	Size varint.VarInt
}

// Returns an Elem which represents the start of an unread EBML stream. Next()
// is the only valid method which can be called on the Elem returned from this
// function (see the package example).
func RootElem(r io.Reader) *Elem {
	return &Elem{
		r:   r,
		buf: bufio.NewReader(r),
	}
}

// Returns the next Elem in the stream. When called on a non-container Elem this
// MUST be called after a data method (e.g. Int(), Bytes(), etc...) has been
// called at least once. For container Elems (and the root Elem) this is the
// only valid method which can be called
func (e *Elem) Next() (*Elem, error) {
	id, err := varint.Read(e.buf)
	if err != nil {
		return nil, err
	}

	size, err := varint.Read(e.buf)
	if err != nil {
		return nil, err
	}

	return &Elem{
		r:    e.r,
		buf:  e.buf,
		Id:   id,
		Size: size,
	}, nil
}

func (e *Elem) fillBuffer() error {
	if e.data == nil {
		size, err := e.Size.Uint64()
		if err != nil {
			return err
		}
		e.data = make([]byte, size)
		if _, err = io.ReadFull(e.buf, e.data); err != nil {
			return err
		}
	}
	return nil
}

// Returns a copy of b padded with zeros on the left so that it matches the
// target size
func leftPad(b []byte, targetSize int) []byte {
	nb := make([]byte, targetSize)
	copy(nb[targetSize-len(b):], b)
	return nb
}

// Reads and returns the Elem's data as a signed integer. This can be called
// multiple times.
func (e *Elem) Int() (int64, error) {
	if e.Size == 0 {
		return 0, nil
	} else if err := e.fillBuffer(); err != nil {
		return 0, err
	}

	var ret int64
	buf := bytes.NewBuffer(leftPad(e.data, 8))
	if err := binary.Read(buf, binary.BigEndian, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

// Reads and returns the Elem's data as an unsigned integer. This can be called
// multiple times.
func (e *Elem) Uint() (uint64, error) {
	if e.Size == 0 {
		return 0, nil
	}
	if err := e.fillBuffer(); err != nil {
		return 0, err
	}

	var ret uint64
	buf := bytes.NewBuffer(leftPad(e.data, 8))
	if err := binary.Read(buf, binary.BigEndian, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

var timeStart = time.Date(
	2001, time.January, 1,
	0, 0, 0, 0,
	time.UTC,
)

// Reads and returns the Elem's data as a Time. This can be called multiple
// times.
func (e *Elem) Date() (time.Time, error) {
	i, err := e.Int()
	if err != nil {
		return time.Time{}, err
	}
	return timeStart.Add(time.Duration(i)), nil
}

// Reads and returns the Elem's data as a float. This can be called multiple
// times.
func (e *Elem) Float() (float64, error) {
	if e.Size == 0 {
		return 0, nil
	} else if e.Size == 4 {
		f, err := e.f32()
		return float64(f), err
	} else if err := e.fillBuffer(); err != nil {
		return 0, err
	}

	var ret float64
	buf := bytes.NewBuffer(leftPad(e.data, 8))
	if err := binary.Read(buf, binary.BigEndian, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (e *Elem) f32() (float32, error) {
	if err := e.fillBuffer(); err != nil {
		return 0, err
	}

	var ret float32
	buf := bytes.NewBuffer(leftPad(e.data, 8))
	if err := binary.Read(buf, binary.BigEndian, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

// Reads and returns the Elem's data as a string. This can be called multiple
// times.
func (e *Elem) Str() (string, error) {
	if e.Size == 0 {
		return "", nil
	} else if err := e.fillBuffer(); err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(e.data)
	ret, err := buf.ReadString(0)
	if err != nil {
		return ret, nil
	} else {
		return ret[:len(ret)-1], nil
	}
}

// Reads and returns the Elem's data as raw bytes. This can be called multiple
// times.
func (e *Elem) Bytes() ([]byte, error) {
	if e.Size == 0 {
		return []byte{}, nil
	} else if err := e.fillBuffer(); err != nil {
		return nil, err
	}

	return e.data, nil
}

// Writes the Elem to the io.Writer as an ebml element. This will only write the
// data portion of the Elem if one of the data methods has been called
// previously. If the Elem is a container it's children will NOT be
// automatically written
func (e *Elem) WriteTo(w io.Writer) (int64, error) {
	var total int64

	i, err := e.Id.WriteTo(w)
	total += int64(i)
	if err != nil {
		return total, err
	}

	i, err = e.Size.WriteTo(w)
	total += int64(i)
	if err != nil {
		return total, err
	}

	if e.data != nil {
		i, err = w.Write(e.data)
		total += int64(i)
		if err != nil {
			return total, err
		}
	}

	return total, nil
}
