package main

import (
	"io"
	"os"
	"github.com/stretchr/testify/require"
	. "testing"

	"github.com/mediocregopher/ebmlstream/edtd"
)

var exampleFiles = []string{
	"test.webm",
	"test-ffmpeg.webm",
}

func TestExampleFiles(t *T) {

	edtdf, err := os.Open("matroska.edtd")
	require.Nil(t, err)

	e, err := edtd.NewEdtd(edtdf)
	require.Nil(t, err)

	for _, fn := range exampleFiles {
		f, err := os.Open(fn)
		require.Nil(t, err, "filename: %s", fn)

		p := e.NewParser(f)
		i := 0
		for {
			_, err := p.Next()
			if err == io.EOF {
				break
			}
			require.Nil(t, err, "filename: %s", fn)
			i++
		}
			require.NotEqual(t,0, i, "filename: %s", fn)
	}
}
