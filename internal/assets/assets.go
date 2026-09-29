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

//go:embed MinecraftRegular.otf
var MinecraftRegularBytes []byte

//go:embed MinecraftBold.otf
var MinecraftBoldBytes []byte

//go:embed MinecraftItalic.otf
var MinecraftItalicBytes []byte

//go:embed MinecraftBoldItalic.otf
var MinecraftBoldItalicBytes []byte

var (
	DefaultIcon         image.Image
	DefaultFont         *opentype.Font // Zpix (CJK pixel font)
	MinecraftRegular    *opentype.Font
	MinecraftBold       *opentype.Font
	MinecraftItalic     *opentype.Font
	MinecraftBoldItalic *opentype.Font
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

	MinecraftRegular, err = opentype.Parse(MinecraftRegularBytes)
	if err != nil {
		panic("failed to parse MinecraftRegular font: " + err.Error())
	}

	MinecraftBold, err = opentype.Parse(MinecraftBoldBytes)
	if err != nil {
		panic("failed to parse MinecraftBold font: " + err.Error())
	}

	MinecraftItalic, err = opentype.Parse(MinecraftItalicBytes)
	if err != nil {
		panic("failed to parse MinecraftItalic font: " + err.Error())
	}

	MinecraftBoldItalic, err = opentype.Parse(MinecraftBoldItalicBytes)
	if err != nil {
		panic("failed to parse MinecraftBoldItalic font: " + err.Error())
	}
}
