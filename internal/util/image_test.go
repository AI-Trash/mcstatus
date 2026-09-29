package util_test

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"net/http/httptest"
	"testing"

	"mcstatus/internal/util"
)

func createTestImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{40, 40, 40, 255}}, image.Point{}, draw.Src)
	for x := 10; x < w-10; x++ {
		for y := 10; y < h-10; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 2), uint8(y * 3), 200, 255})
		}
	}
	return img
}

func TestEncodeImagePNG(t *testing.T) {
	img := createTestImage(64, 64)
	data, mime, err := util.EncodeImage(img, "png")
	if err != nil {
		t.Fatalf("EncodeImage(png) failed: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("MIME want image/png, got %s", mime)
	}
	if len(data) == 0 {
		t.Error("EncodeImage(png) produced empty data")
	}
}

func TestEncodeImageAVIF(t *testing.T) {
	img := createTestImage(64, 64)
	data, mime, err := util.EncodeImage(img, "avif")
	if err != nil {
		t.Fatalf("EncodeImage(avif) failed: %v", err)
	}
	if mime != "image/avif" {
		t.Errorf("MIME want image/avif, got %s", mime)
	}
	if len(data) == 0 {
		t.Error("EncodeImage(avif) produced empty data")
	}

	// Decode back to verify validity
	dec, err := util.DecodeAVIF(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeAVIF failed: %v", err)
	}
	if dec.Bounds().Dx() != 64 || dec.Bounds().Dy() != 64 {
		t.Errorf("Decoded bounds want 64x64, got %v", dec.Bounds())
	}
}

func TestResolveImageFormat(t *testing.T) {
	// 1. Explicit query ?format=avif
	req1 := httptest.NewRequest("GET", "/v2/widget/java/test?format=avif", nil)
	if f := util.ResolveImageFormat(req1); f != "avif" {
		t.Errorf("req1 want avif, got %s", f)
	}

	// 2. Explicit query ?format=png
	req2 := httptest.NewRequest("GET", "/v2/widget/java/test?format=png", nil)
	req2.Header.Set("Accept", "image/avif")
	if f := util.ResolveImageFormat(req2); f != "png" {
		t.Errorf("req2 want png, got %s", f)
	}

	// 3. Accept header image/avif without query param
	req3 := httptest.NewRequest("GET", "/v2/widget/java/test", nil)
	req3.Header.Set("Accept", "image/avif,image/webp,*/*")
	if f := util.ResolveImageFormat(req3); f != "avif" {
		t.Errorf("req3 want avif, got %s", f)
	}

	// 4. Default without query or avif Accept
	req4 := httptest.NewRequest("GET", "/v2/widget/java/test", nil)
	if f := util.ResolveImageFormat(req4); f != "png" {
		t.Errorf("req4 want png, got %s", f)
	}
}

func TestConvertImageBytes(t *testing.T) {
	img := createTestImage(32, 32)
	pngBytes, _, err := util.EncodeImage(img, "png")
	if err != nil {
		t.Fatalf("EncodeImage(png) failed: %v", err)
	}

	// Convert to AVIF
	avifBytes, mime, err := util.ConvertImageBytes(pngBytes, "avif")
	if err != nil {
		t.Fatalf("ConvertImageBytes to avif failed: %v", err)
	}
	if mime != "image/avif" {
		t.Errorf("want image/avif, got %s", mime)
	}
	if len(avifBytes) == 0 {
		t.Error("converted AVIF is empty")
	}

	// Passthrough PNG
	sameBytes, sameMime, err := util.ConvertImageBytes(pngBytes, "png")
	if err != nil {
		t.Fatalf("ConvertImageBytes to png failed: %v", err)
	}
	if sameMime != "image/png" || len(sameBytes) != len(pngBytes) {
		t.Errorf("PNG passthrough unexpected result")
	}
}
func TestDecodeBase64(t *testing.T) {
	// Standard data URI
	dataUri := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	raw, err := util.DecodeBase64Bytes(dataUri)
	if err != nil || len(raw) == 0 {
		t.Fatalf("DecodeBase64Bytes failed: %v", err)
	}

	img, err := util.DecodeBase64Image(dataUri)
	if err != nil || img == nil {
		t.Fatalf("DecodeBase64Image failed: %v", err)
	}

	// Empty
	if _, err := util.DecodeBase64Bytes(""); err == nil {
		t.Error("expected error on empty string")
	}
}
