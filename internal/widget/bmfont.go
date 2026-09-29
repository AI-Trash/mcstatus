package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/fzipp/bmfont"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"mcstatus/internal/assets"
)

var (
	mcBMFont *bmfont.BitmapFont
	mcBMMask *image.Alpha
	mcWidths [256]int
)

func initBMFont() {
	if assets.MinecraftASCIIImage == nil {
		return
	}

	srcImg := assets.MinecraftASCIIImage
	mcBMMask = image.NewAlpha(image.Rect(0, 0, 256, 256))

	for ch := 0; ch < 256; ch++ {
		col := ch % 16
		row := ch / 16
		rightmost := -1
		for x := 7; x >= 0; x-- {
			for y := 0; y < 8; y++ {
				if _, _, _, a := srcImg.At(col*8+x, row*8+y).RGBA(); a > 0 {
					rightmost = x
					break
				}
			}
			if rightmost != -1 {
				break
			}
		}
		if ch == 32 {
			mcWidths[ch] = 4
		} else if rightmost == -1 {
			mcWidths[ch] = 0
		} else {
			mcWidths[ch] = rightmost + 2
		}

		// Scale 2x into 256x256 Alpha mask
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if _, _, _, a := srcImg.At(col*8+x, row*8+y).RGBA(); a > 0 {
					mcBMMask.SetAlpha((col*8+x)*2, (row*8+y)*2, color.Alpha{A: 255})
					mcBMMask.SetAlpha((col*8+x)*2+1, (row*8+y)*2, color.Alpha{A: 255})
					mcBMMask.SetAlpha((col*8+x)*2, (row*8+y)*2+1, color.Alpha{A: 255})
					mcBMMask.SetAlpha((col*8+x)*2+1, (row*8+y)*2+1, color.Alpha{A: 255})
				}
			}
		}
	}

	sheetFunc := func(filename string) (io.ReadCloser, error) {
		var b bytes.Buffer
		_ = png.Encode(&b, mcBMMask)
		return io.NopCloser(&b), nil
	}

	var err error
	mcBMFont, err = bmfont.Read(bytes.NewReader(assets.MinecraftFNTBytes), sheetFunc)
	if err != nil {
		panic("failed to parse minecraft bmfont: " + err.Error())
	}
}

// BMFontFace wraps bmfont.BitmapFont into Go's standard font.Face.
type BMFontFace struct {
	Font  *bmfont.BitmapFont
	Sheet image.Image
	Bold  bool
}

func (f *BMFontFace) Close() error                     { return nil }
func (f *BMFontFace) PixelsPerInch() float64           { return 72.0 }
func (f *BMFontFace) Kern(r0, r1 rune) fixed.Int26_6   { return 0 }
func (f *BMFontFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:  fixed.I(16),
		Ascent:  fixed.I(14),
		Descent: fixed.I(2),
	}
}

func (f *BMFontFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	if r < 0 || r >= 256 {
		return fixed.I(8), false
	}
	adv := mcWidths[r] * 2
	if f.Bold {
		adv += 2
	}
	return fixed.I(adv), true
}

func (f *BMFontFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	adv, ok := f.GlyphAdvance(r)
	bounds := fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: 0, Y: -fixed.I(14)},
		Max: fixed.Point26_6{X: adv, Y: fixed.I(2)},
	}
	return bounds, adv, ok
}

func (f *BMFontFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	c, ok := f.Font.Descriptor.Chars[r]
	if !ok {
		c = f.Font.Descriptor.Chars['?']
	}
	minX := dot.X.Floor() + c.XOffset
	minY := dot.Y.Floor() - 14 + c.YOffset
	dr := image.Rect(minX, minY, minX+c.Width, minY+c.Height)
	sp := image.Pt(c.X, c.Y)
	adv, _ := f.GlyphAdvance(r)
	return dr, f.Sheet, sp, adv, true
}
