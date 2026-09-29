package assets

import (
	"archive/tar"
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
//go:embed zpix.ttf.zst
var zpixZstBytes []byte

//go:embed unifont.otf.zst
var unifontZstBytes []byte

//go:embed minecraft_family.tar.zst
var minecraftFamilyTarZstBytes []byte

//go:embed ascii.png
var MinecraftASCIIBytes []byte

//go:embed minecraft.fnt
var MinecraftFNTBytes []byte

var (
	DefaultIcon           image.Image
	DefaultFont           *opentype.Font // Zpix (CJK pixel font)
	MinecraftRegular      *opentype.Font
	MinecraftBold         *opentype.Font
	MinecraftItalic       *opentype.Font
	MinecraftBoldItalic   *opentype.Font
	Unifont               *opentype.Font // GNU Unifont (Official Minecraft Java fallback)
	MinecraftASCIIImage   image.Image
	MinecraftRegularBytes []byte
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

	// Decompress Minecraft font family from single solid archive
	tarBytes, err := decompressZstd(minecraftFamilyTarZstBytes)
	if err != nil {
		panic("failed to decompress minecraft_family.tar.zst: " + err.Error())
	}
	tr := tar.NewReader(bytes.NewReader(tarBytes))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic("failed to read tar entry: " + err.Error())
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			panic("failed to read font data: " + err.Error())
		}

		switch hdr.Name {
		case "MinecraftRegular.otf":
			MinecraftRegularBytes = data
			MinecraftRegular, _ = opentype.Parse(data)
		case "MinecraftBold.otf":
			MinecraftBold, _ = opentype.Parse(data)
		case "MinecraftItalic.otf":
			MinecraftItalic, _ = opentype.Parse(data)
		case "MinecraftBoldItalic.otf":
			MinecraftBoldItalic, _ = opentype.Parse(data)
		}
	}
}
