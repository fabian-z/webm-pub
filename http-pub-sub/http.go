package httppubsub

import (
	"bufio"
	"io"
	"net/http"
	"time"

	"github.com/fabian-z/webm-pub/http-pub-sub/pubsub"
)

// Defines an HTTPPubSub app. Usage is based on using an instance from
// DefaultHTTPPubSubApp() and replacing individual functions in it with your own
// for where you want to change behavior. This modified instance is then passed
// into NewHTTPPubSub.
//
// All requests can return an integer and a string. If the int is non-zero than
// the request will be cut-short and that response code will be sent back, along
// with the returned string as the response body.
type HTTPPubSubApp struct {
	// Do whatever needs to be done upon the opening of a client connection
	// (either POST or GET). Returns the name of the channel and a piece of
	// state which will passed into all other functions for the app and only
	// lives as long as the request.  Default state is nil.
	OnOpen func(http.ResponseWriter, *http.Request) (
		string, interface{}, int, string)

	// Returns the next slice of data from the publisher to be sent to all the
	// subscribers. A non-zero return will send the returned data to the
	// subscribers and that code will be returned to the publisher
	GetNext func(interface{}, *http.Request) ([]byte, int, string)

	// Do whatever needs to be done upon a request closing, either being closed
	// by the client or by the server. This is called as the very last step of a
	// request, after all other cleanup and writing to the client is done. It is
	// not called if the client was killed by a previous app method cutting it
	// short.
	//
	// Note that it is not necessary to close the request body here, that will
	// be done by the package
	OnClose func(interface{}, http.ResponseWriter, *http.Request) (int, string)
} 

// Returns an HTTPPubSubApp with all default behavior, which can then be chaned
// on a function-by-function basis as needed
func DefaultHTTPPubSubApp() *HTTPPubSubApp {
	return &HTTPPubSubApp{
		OnOpen: func(
			w http.ResponseWriter, req *http.Request,
		) (
			string, interface{}, int, string,
		) {
			return req.URL.Path, nil, 0, ""
		},

		GetNext: func(s interface{}, req *http.Request) ([]byte, int, string) {
			b := make([]byte, 10240)
			_, err := io.ReadFull(req.Body, b)
			if err == io.EOF {
				return b, 200, ""
			} else {
				return b, 500, "unknown error reading"
			}
			return b, 0, ""
		},

		OnClose: func(
			s interface{},  w http.ResponseWriter, req *http.Request,
		) (
			int, string,
		) {
			return 0, ""
		},
	}
}

// An instance of HTTPPubSub. Can be used either standolone or as a handler for
// an existing http.Server. When a client POSTs, the data for that POST is
// forwarded to all clients GETing that same channel (which is based on the
// path). The POST can be long-running or short-lived. Two clients can not POST
// to the same channel at the same time.
type HTTPPubSub struct {
	pmux *pubsub.PubSubMux
	app *HTTPPubSubApp
}

func NewHTTPPubSub(app *HTTPPubSubApp) *HTTPPubSub {
	return &HTTPPubSub{
		pmux: pubsub.NewPubSubMux(),
		app: app,
	}
}

func bail(w http.ResponseWriter, code int, ret string) bool {
	if code == 0 {
		return false
	}

	w.WriteHeader(code)
	w.Write([]byte(ret))
	return true
}

func (h *HTTPPubSub) doLast(
	state interface{}, w http.ResponseWriter, req *http.Request,
) {
	code, ret := h.app.OnClose(state, w, req)
	bail(w, code, ret)
}

// Implements ServeHTTP for the http.Handler interface
func (h *HTTPPubSub) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	channel, state, code, ret := h.app.OnOpen(w, req)
	if bail(w, code, ret) {
		return
	}

	if req.Method == "POST" {

		ch, exists := h.pmux.GetPubCh(channel)
		if exists {
			w.WriteHeader(403)
			h.doLast(state, w, req)
			return
		}
		defer h.pmux.ClosePubCh(channel)

		for {
			b, code, ret := h.app.GetNext(state, req)
			if b != nil && len(b) > 0 {
				select {
				case ch <- b:
				case <-time.After(5 * time.Second):
				}
			}
			if bail(w, code, ret) {
				return
			}
		}

	} else if req.Method == "GET" {

		ch := make(chan interface{}, 100)
		h.pmux.AddSubCh(channel, ch)
		buf := bufio.NewWriter(w)

		for bi := range ch {
			b := bi.([]byte)
			_, err := buf.Write(b)
			if err != nil {
				h.pmux.RemSubCh(channel, ch)
				h.doLast(state, w, req)
				return
			}
			buf.Flush()
		}
	}

	// We can't just defer this, because if an app method returns an error we
	// don't want to call it.
	h.doLast(state, w, req)
}

// Shortcut for calling:
//	s := http.NewServeMux()
//	s.Handle("/", httpPubSubInstance)
//	http.ListenAndServe(addr, s)
func (h *HTTPPubSub) ListenAndServe(addr string) error {
	s := http.NewServeMux()
	s.Handle("/", h)
	return http.ListenAndServe(addr, s)
}
