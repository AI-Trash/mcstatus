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
}

// DecodeAVIF reads an AVIF image from r.
func DecodeAVIF(r io.Reader) (image.Image, error) {
	return gavif.Decode(r)
}

// DecodeAVIFConfig returns color model and dimensions of an AVIF image.
func DecodeAVIFConfig(r io.Reader) (image.Config, error) {
	return gavif.DecodeConfig(r)
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

// EncodeImage encodes an image.Image into target format ("avif" or "png").
// It returns the encoded bytes, the MIME Content-Type, and any error.
func EncodeImage(img image.Image, format string) ([]byte, string, error) {
	var buf bytes.Buffer
	format = strings.ToLower(strings.TrimSpace(format))

	if format == "avif" {
		if err := EncodeAVIF(&buf, img, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/avif", nil
	}

	// Default to standard PNG
	if err := png.Encode(&buf, img); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/png", nil
}

// ResolveImageFormat determines whether to serve "avif" or "png" based on
// explicit ?format= query param, falling back to HTTP Accept header if not specified.
func ResolveImageFormat(r *http.Request) string {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "avif" || format == "png" {
		return format
	}

	// If no explicit query param, check Accept header
	if strings.Contains(r.Header.Get("Accept"), "image/avif") {
		return "avif"
	}

	return "png"
}

// ConvertImageBytes converts existing image bytes (such as PNG) to the target format if needed.
func ConvertImageBytes(raw []byte, targetFormat string) ([]byte, string, error) {
	targetFormat = strings.ToLower(strings.TrimSpace(targetFormat))
	if targetFormat != "avif" {
		return raw, "image/png", nil
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	return EncodeImage(img, "avif")
}

// DecodeBase64Image decodes a base64-encoded image string (Data URI or raw base64).
func DecodeBase64Image(s string) (image.Image, error) {
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
		if err != nil {
			return nil, err
		}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
