package main

import (
	"log"
	"net/http"
	"sync"

	hps "github.com/fabian-z/webm-pub/http-pub-sub"
	"github.com/fabian-z/webm-pub/webmkeeper"
)

type connState struct {
	isBcaster bool
	channel   string
	keeper    *webmkeeper.WebmKeeper
}

var keepers = map[string]*webmkeeper.WebmKeeper{}
var keepersLock sync.RWMutex

func getKeeper(channel string) (*webmkeeper.WebmKeeper, bool) {
	keepersLock.RLock()
	defer keepersLock.RUnlock()
	k, ok := keepers[channel]
	return k, ok
}

func addKeeper(channel string, k *webmkeeper.WebmKeeper) bool {
	keepersLock.Lock()
	defer keepersLock.Unlock()
	if _, ok := keepers[channel]; ok {
		return false
	}
	keepers[channel] = k
	return true
}

func remKeeper(channel string) {
	keepersLock.Lock()
	defer keepersLock.Unlock()
	delete(keepers, channel)
}

func main() {
	app := hps.DefaultHTTPPubSubApp()

	app.OnOpen =
		func(w http.ResponseWriter, req *http.Request) (string, interface{}, int, string) {
			channel := req.URL.Path
			log.Printf("%s to %s opened", req.Method, channel)
			var s connState
			s.channel = channel

			if req.Method == "POST" {
				s.isBcaster = true
				k, err := webmkeeper.New(req.Body)
				if err != nil {
					log.Printf("%s has error on open: %s", channel, err)
					return channel, s, 400, err.Error()
				}
				s.keeper = k

				if !addKeeper(channel, k) {
					return channel, s, 400, "has a writer already"
				}
			} else {
				k, ok := getKeeper(channel)
				if !ok {
					return channel, s, 404, "couldn't find stream "+channel
				}
				k.Bootstrap(w)
			}

			return channel, s, 0, ""
		}

	app.GetNext =
		func(s interface{}, req *http.Request) ([]byte, int, string) {
			b, err := s.(connState).keeper.Next()
			if err != nil {
				channel := s.(connState).channel
				log.Printf("reading from %s: %s", channel, err)
				remKeeper(channel)
				return b, 500, err.Error()
			}
			return b, 0, ""
		}

	app.OnClose =
		func(s interface{}, w http.ResponseWriter, req *http.Request) (int, string) {
			channel := s.(connState).channel
			log.Printf("%s to %s closed", req.Method, channel)
			if s.(connState).isBcaster {
				remKeeper(channel)
			}
			return 0, ""
		}

	addr := ":8090"

	h := hps.NewHTTPPubSub(app)
	http.Handle("/stream/", h)

	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
