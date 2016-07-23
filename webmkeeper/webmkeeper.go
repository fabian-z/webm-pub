// Reads in a webm stream and keeps track of the data such that a new client can
// come in mid-way through the stream and receive correctly formed data
package webmkeeper

import (
	"bytes"
	"fmt"
	"github.com/mediocregopher/ebmlstream/edtd"
	"io"

	"github.com/fabian-z/webm-pub/webm"
)

func next(p *edtd.Parser) (*edtd.Elem, error) {
	skipLevel := uint64(0)
	for {
		el, err := p.Next()
		if err != nil {
			return el, err
		}

		if skipLevel > 0 && el.Level > skipLevel { 
			continue
		}
		if skipLevel == el.Level {
			skipLevel = 0
		}
		//switch el.Name {
		//case "Void":
		//	continue
		//case "SeekHead":
		//	skipLevel = el.Level
		//	continue
		//}
		return el, nil
	}
}

type WebmKeeper struct {
	p           *edtd.Parser
	header      *bytes.Buffer
	body        *bytes.Buffer
	elemBuf     *bytes.Buffer

	// fields for tracking random access points
	lastCluster        int
	trackBlockCount    [2]byte
	trackBlockKeyframe [2]bool
}

// Reads the ebml and Segment header portions of the webm stream so that they
// are available immediately, then returns a WebmKeeper ready for reading and
// writing
func New(r io.Reader) (*WebmKeeper, error) {
	p := webmEdtd.NewParser(r)
	header := bytes.NewBuffer(make([]byte, 0, 4096))
	body := bytes.NewBuffer(make([]byte, 0, 4096))

	for {
		el, err := next(p)
		if err != nil {
			return nil, err
		}

		if el.Name == "Cluster" {
			el.WriteTo(body)
			break
		}
		el.WriteTo(header)
	}

	return &WebmKeeper{
		p:       p,
		header:  header,
		body:    body,
		elemBuf: bytes.NewBuffer(make([]byte, 0, 1024)),
	}, nil
}


// Given an io.Writer which is new to the stream, sends it the header and
// current body data necessary for it to catch up and begin playing a valid
// stream
func (wk *WebmKeeper) Bootstrap(w io.Writer) error {
	if _, err := w.Write(wk.header.Bytes()); err != nil {
		return err
	}
	if _, err := w.Write(wk.body.Bytes()); err != nil {
		return err
	}
	return nil
}

// Returns the next piece of the stream available for writing to clients.
func (wk *WebmKeeper) Next() ([]byte, error) {
	el, err := next(wk.p)
	if err != nil {
		return nil, err
	}

	if err = wk.handleRandomAccessPoint(el); err != nil {
		return nil, err
	}

	wk.elemBuf.Reset()
	el.WriteTo(wk.elemBuf)
	b := make([]byte, wk.elemBuf.Len())
	copy(b, wk.elemBuf.Bytes())

	wk.body.Write(b)
	return b, nil
}

// This will handle making sure that wk.body always starts at the most recent
// random access point we've seen. It expects to be run before the given element
// has actually been written to the buffer
func (wk *WebmKeeper) handleRandomAccessPoint(el *edtd.Elem) error {
	switch el.Name {
	case "Cluster":
		wk.lastCluster = wk.body.Len()
		wk.resetTracking()
		return nil

	case "SimpleBlock", "Block":

		trackNumber, keyframe, err := webm.AsKeyBlock(el)
		if err != nil {
			return err
		}
		if trackNumber > 2 || trackNumber < 1 {
			return fmt.Errorf("impossible track number %d", trackNumber)
		}
		trackNumber--

		wk.incTrackCount(trackNumber)

		if keyframe {
			wk.trackBlockKeyframe[trackNumber] = true
			b := make([]byte, wk.body.Len() - wk.lastCluster)
			copy(b, wk.body.Bytes()[wk.lastCluster:])
			wk.body.Reset()
			wk.body.Write(b)
			wk.lastCluster = 0
		}

		return nil

	default:
		return nil
	}
}

func (wk *WebmKeeper) resetTracking() {
	wk.trackBlockCount    = [2]byte{}
	wk.trackBlockKeyframe = [2]bool{}
}

func (wk *WebmKeeper) incTrackCount(trackNumber uint64) {
	if wk.trackBlockCount[trackNumber] < 255 {
		wk.trackBlockCount[trackNumber]++
	}
}
