package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mediocregopher/ebmlstream/edtd"
)

func main() {
	edtdf, err := os.Open("matroska.edtd")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("parsing matroska.edtd")
	e, err := edtd.NewEdtd(edtdf)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting parswer for %s", os.Args[1])
	p := e.NewParser(f)
	for {
		el, err := p.Next()
		if err != nil {
			log.Fatal(err)
		}

		size64, err := el.Elem.Size.Uint64()
		if err != nil {
			log.Fatal(err)
		}

		tabs := strings.Repeat("\t", int(el.Level))
		prefix := fmt.Sprintf(
			"%s%s 0x%x [size: %d | 0x%x]",
			tabs, el.Name, el.Elem.Id, size64, el.Elem.Size,
		)
		var line string
		var thing interface{}
		switch el.Type {
		case edtd.Int:
			thing, err = el.Int()
			line = fmt.Sprintf("%s - %d", prefix, thing)
		case edtd.Uint:
			thing, err = el.Uint()
			line = fmt.Sprintf("%s - %d", prefix, thing)
		case edtd.Float:
			thing, err = el.Float()
			line = fmt.Sprintf("%s - %f", prefix, thing)
		case edtd.String:
			thing, err = el.Str()
			line = fmt.Sprintf("%s - %s", prefix, thing)
		default:
			line = prefix
		}

		if err != nil {
			log.Fatalf("line: %s, err: %s", line, err)
		}

		log.Println(line)
	}
}
