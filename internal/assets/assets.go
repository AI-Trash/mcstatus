package assets

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"io"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

//go:embed default_icon.png
var DefaultIconBytes []byte

//go:embed zpix.ttf.zst
var zpixZstBytes []byte

//go:embed unifont.otf.zst
var unifontZstBytes []byte

//go:embed minecraft.otc.zst
var minecraftOTCZstBytes []byte

//go:embed minecraft.bdf.zst
var MinecraftBDFZstBytes []byte

var (
	DefaultIcon         image.Image
	DefaultFont         *opentype.Font // Zpix (CJK pixel font)
	MinecraftRegular    *sfnt.Font
	MinecraftBold       *sfnt.Font
	MinecraftItalic     *sfnt.Font
	MinecraftBoldItalic *sfnt.Font
	Unifont             *opentype.Font // GNU Unifont (Official Minecraft Java fallback)
	MinecraftBDFBytes   []byte
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

	MinecraftBDFBytes, err = decompressZstd(MinecraftBDFZstBytes)
	if err != nil {
		panic("failed to decompress minecraft.bdf.zst: " + err.Error())
	}

	zpixBytes, err := decompressZstd(zpixZstBytes)
	if err != nil {
		panic("failed to decompress zpix.ttf.zst: " + err.Error())
	}
	DefaultFont, err = opentype.Parse(zpixBytes)
	if err != nil {
		panic("failed to parse zpix font: " + err.Error())
	}

	unifontBytes, err := decompressZstd(unifontZstBytes)
	if err != nil {
		panic("failed to decompress unifont.otf.zst: " + err.Error())
	}
	Unifont, err = opentype.Parse(unifontBytes)
	if err != nil {
		panic("failed to parse unifont: " + err.Error())
	}

	otcBytes, err := decompressZstd(minecraftOTCZstBytes)
	if err != nil {
		panic("failed to decompress minecraft.otc.zst: " + err.Error())
	}
	coll, err := opentype.ParseCollection(otcBytes)
	if err != nil {
		panic("failed to parse minecraft.otc: " + err.Error())
	}
	if coll.NumFonts() >= 4 {
		MinecraftRegular, _ = coll.Font(0)
		MinecraftBold, _ = coll.Font(1)
		MinecraftItalic, _ = coll.Font(2)
		MinecraftBoldItalic, _ = coll.Font(3)
	}
}
