package widget

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/KarpelesLab/goavif"
)

func TestCompressionComparison(t *testing.T) {
	d := widgetDataForTest()
	img, err := RenderMinecraftImage(&d)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// 1. Default PNG
	var defBuf bytes.Buffer
	if err := png.Encode(&defBuf, img); err != nil {
		t.Fatalf("PNG encode failed: %v", err)
	}

	// 2. Lossless AVIF (Speed: 6)
	var avifBuf bytes.Buffer
	err = goavif.Encode(&avifBuf, img, &goavif.Options{
		Lossless: true,
		Speed:    6,
	})
	if err != nil {
		t.Fatalf("AVIF encode failed: %v", err)
	}

	if defBuf.Len() == 0 {
		t.Fatal("PNG produced empty output")
	}
	if avifBuf.Len() == 0 {
		t.Fatal("AVIF produced empty output")
	}

	// Verify RenderFormatted API
	d.Format = "avif"
	avifBytes, mime, err := RenderFormatted(&d)
	if err != nil || mime != "image/avif" || len(avifBytes) == 0 {
		t.Fatalf("RenderFormatted(avif) failed: mime=%s, err=%v", mime, err)
	}

	d.Format = "png"
	pngBytes, mime, err := RenderFormatted(&d)
	if err != nil || mime != "image/png" || len(pngBytes) == 0 {
		t.Fatalf("RenderFormatted(png) failed: mime=%s, err=%v", mime, err)
	}
}

func widgetDataForTest() WidgetData {
	return WidgetData{
		Style:         "minecraft",
		Online:        true,
		Host:          "pure.mc.asyncraft.club",
		Port:          25565,
		Edition:       "Java Edition",
		Version:       "Velocity 1.7.2-26.2",
		PlayersOnline: 1,
		PlayersMax:    500,
		MOTDRaw:       "§#55ffffA§#74ffffs§#93ffffy§#b2ffffn§#d1ffffc§#f0ffffra§#d1fffff§#b2fffft§#93ffff服务器\n §7-> §a26.2纯净",
		Dark:          true,
		Rounded:       true,
	}
}
