package assets

import (
	_ "embed"
	"image"
	"image/png"
	"bytes"
)

//go:embed default_icon.png
var DefaultIconBytes []byte

var DefaultIcon image.Image

func init() {
	var err error
	DefaultIcon, err = png.Decode(bytes.NewReader(DefaultIconBytes))
	if err != nil {
		panic("failed to decode embedded default icon: " + err.Error())
	}
}
