package assets

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"image"
	"image/png"
	"io"

	"golang.org/x/image/font/opentype"
)

//go:embed default_icon.png
var DefaultIconBytes []byte

//go:embed font.ttf.gz
var fontGzBytes []byte

var (
	DefaultIcon image.Image
	DefaultFont *opentype.Font
)

func init() {
	var err error
	DefaultIcon, err = png.Decode(bytes.NewReader(DefaultIconBytes))
	if err != nil {
		panic("failed to decode embedded default icon: " + err.Error())
	}

	gr, err := gzip.NewReader(bytes.NewReader(fontGzBytes))
	if err != nil {
		panic("failed to decompress font: " + err.Error())
	}
	fontBytes, err := io.ReadAll(gr)
	if err != nil {
		panic("failed to read decompressed font: " + err.Error())
	}
	_ = gr.Close()

	DefaultFont, err = opentype.Parse(fontBytes)
	if err != nil {
		panic("failed to parse font: " + err.Error())
	}
}
