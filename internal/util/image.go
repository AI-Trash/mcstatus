package util

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	gavif "github.com/gen2brain/gav1d/avif"
	"github.com/deepteams/webp"
)
// AVIFOptions defines AVIF encoding parameters.
type AVIFOptions struct {
	Quality      int
	QualityAlpha int
	Speed        int
	Lossless     bool
}

// DefaultAVIFOptions specifies default AVIF encoding parameters.
var DefaultAVIFOptions = AVIFOptions{
	Quality:      100,
	QualityAlpha: 100,
	Speed:        6,
	Lossless:     true,
}

func init() {
	image.RegisterFormat("avif", "????ftypavif", DecodeAVIF, DecodeAVIFConfig)
	image.RegisterFormat("avif", "????ftypavis", DecodeAVIF, DecodeAVIFConfig)
	image.RegisterFormat("avif", "????ftypmif1", DecodeAVIF, DecodeAVIFConfig)
	image.RegisterFormat("webp", "RIFF????WEBP", DecodeWebP, DecodeWebPConfig)
}
// DecodeAVIF reads an AVIF image from r.
func DecodeAVIF(r io.Reader) (image.Image, error) {
	return gavif.Decode(r)
}

// DecodeAVIFConfig returns color model and dimensions of an AVIF image.
func DecodeAVIFConfig(r io.Reader) (image.Config, error) {
	return gavif.DecodeConfig(r)
}
// WebPOptions defines lossless WebP encoding parameters.
type WebPOptions struct {
	Lossless bool
	Method   int
}

// DefaultWebPOptions specifies default lossless WebP encoding parameters.
var DefaultWebPOptions = WebPOptions{
	Lossless: true,
	Method:   4,
}

// DecodeWebP reads a WebP image from r.
func DecodeWebP(r io.Reader) (image.Image, error) {
	return webp.Decode(r)
}

// DecodeWebPConfig returns dimensions of a WebP image.
func DecodeWebPConfig(r io.Reader) (image.Config, error) {
	return webp.DecodeConfig(r)
}

// EncodeWebP writes m to w in WebP format.
func EncodeWebP(w io.Writer, m image.Image, opts *WebPOptions) error {
	o := DefaultWebPOptions
	if opts != nil {
		o = *opts
	}
	return webp.Encode(w, m, &webp.Options{
		Lossless: o.Lossless,
		Method:   o.Method,
	})
}
// EncodeAVIF writes m to w in AVIF format.
func EncodeAVIF(w io.Writer, m image.Image, opts *AVIFOptions) error {
	o := DefaultAVIFOptions
	if opts != nil {
		o = *opts
		if o.Quality <= 0 {
			o.Quality = DefaultAVIFOptions.Quality
		}
		if o.QualityAlpha <= 0 {
			o.QualityAlpha = o.Quality
		}
		if o.Speed < 0 {
			o.Speed = DefaultAVIFOptions.Speed
		}
	}
	return gavif.Encode(w, m, gavif.EncodeOptions{
		Quality:      o.Quality,
		QualityAlpha: o.QualityAlpha,
		Speed:        o.Speed,
		Lossless:     o.Lossless,
	})
}
// EncodeImage encodes an image.Image into target format ("webp", "png", or "avif").
// Default format is "webp". It directly encodes and serves the specified format without fallback.
func EncodeImage(img image.Image, format string) ([]byte, string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "webp"
	}

	var buf bytes.Buffer
	switch format {
	case "webp":
		if err := EncodeWebP(&buf, img, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil

	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil

	case "avif":
		if err := EncodeAVIF(&buf, img, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/avif", nil

	default:
		if err := EncodeWebP(&buf, img, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil
	}
}

// ResolveImageFormat determines the image format to serve based on
// explicit ?format= query param, falling back to HTTP Accept header,
// with the default format being "webp".
func ResolveImageFormat(r *http.Request) string {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "webp" || format == "png" || format == "avif" {
		return format
	}

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "image/webp") {
		return "webp"
	}
	if strings.Contains(accept, "image/avif") {
		return "avif"
	}
	if strings.Contains(accept, "image/png") && !strings.Contains(accept, "*/*") {
		return "png"
	}

	return "webp"
}
// ConvertImageBytes converts existing image bytes (such as PNG) to the target format.
func ConvertImageBytes(raw []byte, targetFormat string) ([]byte, string, error) {
	targetFormat = strings.ToLower(strings.TrimSpace(targetFormat))
	if targetFormat == "" {
		targetFormat = "webp"
	}

	// If source is already PNG and target is PNG, return as-is
	if targetFormat == "png" {
		return raw, "image/png", nil
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	return EncodeImage(img, targetFormat)
}
// DecodeBase64Bytes decodes a base64-encoded string (Data URI or raw base64) into raw bytes.
func DecodeBase64Bytes(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty base64 string")
	}
	if idx := strings.Index(s, ","); idx != -1 {
		s = s[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(s)
	}
	return data, err
}

// DecodeBase64Image decodes a base64-encoded image string (Data URI or raw base64).
func DecodeBase64Image(s string) (image.Image, error) {
	data, err := DecodeBase64Bytes(s)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
