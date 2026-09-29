package assets

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"io"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/image/font/opentype"
)

//go:embed default_icon.png
var DefaultIconBytes []byte

//go:embed font.ttf.zst
var fontZstBytes []byte

//go:embed unifont.otf.zst
var unifontZstBytes []byte

//go:embed MinecraftRegular.otf
var MinecraftRegularBytes []byte

//go:embed MinecraftBold.otf
var MinecraftBoldBytes []byte

//go:embed MinecraftItalic.otf
var MinecraftItalicBytes []byte

//go:embed MinecraftBoldItalic.otf
var MinecraftBoldItalicBytes []byte

//go:embed ascii.png
var MinecraftASCIIBytes []byte

//go:embed minecraft.fnt
var MinecraftFNTBytes []byte

var (
	DefaultIcon         image.Image
	DefaultFont         *opentype.Font // Zpix (CJK pixel font)
	MinecraftRegular    *opentype.Font
	MinecraftBold       *opentype.Font
	MinecraftItalic     *opentype.Font
	MinecraftBoldItalic *opentype.Font
	Unifont             *opentype.Font // GNU Unifont (Official Minecraft Java fallback)
	MinecraftASCIIImage image.Image
)

func decompressZstd(data []byte) ([]byte, error) {
	zr, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func init() {
	var err error
	DefaultIcon, err = png.Decode(bytes.NewReader(DefaultIconBytes))
	if err != nil {
		panic("failed to decode embedded default icon: " + err.Error())
	}

	MinecraftASCIIImage, _, err = image.Decode(bytes.NewReader(MinecraftASCIIBytes))
	if err != nil {
		panic("failed to decode embedded ascii.png: " + err.Error())
	}

	fontBytes, err := decompressZstd(fontZstBytes)
	if err != nil {
		panic("failed to decompress font.ttf.zst: " + err.Error())
	}
	DefaultFont, err = opentype.Parse(fontBytes)
	if err != nil {
		panic("failed to parse font: " + err.Error())
	}

	unifontBytes, err := decompressZstd(unifontZstBytes)
	if err != nil {
		panic("failed to decompress unifont.otf.zst: " + err.Error())
	}
	Unifont, err = opentype.Parse(unifontBytes)
	if err != nil {
		panic("failed to parse unifont: " + err.Error())
	}

	if len(MinecraftRegularBytes) > 0 {
		MinecraftRegular, _ = opentype.Parse(MinecraftRegularBytes)
	}
	if len(MinecraftBoldBytes) > 0 {
		MinecraftBold, _ = opentype.Parse(MinecraftBoldBytes)
	}
	if len(MinecraftItalicBytes) > 0 {
		MinecraftItalic, _ = opentype.Parse(MinecraftItalicBytes)
	}
	if len(MinecraftBoldItalicBytes) > 0 {
		MinecraftBoldItalic, _ = opentype.Parse(MinecraftBoldItalicBytes)
	}
}
