package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"

	"mcstatus/internal/assets"
)

// WidgetData holds all parameters needed to render a server status widget banner or card.
type WidgetData struct {
	Style         string      // "default", "banner", or "minecraft"
	Online        bool
	Host          string
	Port          uint16
	Edition       string      // ("Java Edition" or "Bedrock Edition")
	Version       string
	PlayersOnline int
	PlayersMax    int
	MOTD          string      // clean text
	MOTDRaw       string      // raw MOTD text (with formatting codes)
	Icon          image.Image // if nil, uses assets.DefaultIcon
	Dark          bool        // default true
	Rounded       bool        // default true
	Transparent   bool        // default false
	HideIcon      bool        // hide with ?icon=false
	HideAddress   bool        // hide with ?address=false or ?title=false
}
var (
	// Default banner fonts
	titleFace  font.Face
	normalFace font.Face

	// Minecraft in-game style fonts
	mcRegularFace    font.Face
	mcBoldFace       font.Face
	mcItalicFace     font.Face
	mcBoldItalicFace font.Face
	mcCJKFace        font.Face
	mcSfntFont       *sfnt.Font
)

func init() {
	// 1. Initialize default banner fonts
	if assets.DefaultFont != nil {
		var err error
		titleFace, err = opentype.NewFace(assets.DefaultFont, &opentype.FaceOptions{
			Size: 20,
			DPI:  72,
		})
		if err != nil {
			titleFace = basicfont.Face7x13
		}
		normalFace, err = opentype.NewFace(assets.DefaultFont, &opentype.FaceOptions{
			Size: 12,
			DPI:  72,
		})
		if err != nil {
			normalFace = basicfont.Face7x13
		}
	} else {
		titleFace = basicfont.Face7x13
		normalFace = basicfont.Face7x13
	}

	// 2. Initialize Minecraft pixel fonts (Size: 16, DPI: 72)
	mcOpts := &opentype.FaceOptions{
		Size: 16,
		DPI:  72,
	}

	if assets.MinecraftRegular != nil {
		if f, err := opentype.NewFace(assets.MinecraftRegular, mcOpts); err == nil {
			mcRegularFace = f
		}
	}
	if assets.MinecraftBold != nil {
		if f, err := opentype.NewFace(assets.MinecraftBold, mcOpts); err == nil {
			mcBoldFace = f
		}
	}
	if assets.MinecraftItalic != nil {
		if f, err := opentype.NewFace(assets.MinecraftItalic, mcOpts); err == nil {
			mcItalicFace = f
		}
	}
	if assets.MinecraftBoldItalic != nil {
		if f, err := opentype.NewFace(assets.MinecraftBoldItalic, mcOpts); err == nil {
			mcBoldItalicFace = f
		}
	}
	if assets.Unifont != nil {
		if f, err := opentype.NewFace(assets.Unifont, mcOpts); err == nil {
			mcCJKFace = f
		}
	} else if assets.DefaultFont != nil {
		if f, err := opentype.NewFace(assets.DefaultFont, mcOpts); err == nil {
			mcCJKFace = f
		}
	}

	// Fallbacks
	if mcRegularFace == nil {
		mcRegularFace = basicfont.Face7x13
	}
	if mcBoldFace == nil {
		mcBoldFace = mcRegularFace
	}
	if mcItalicFace == nil {
		mcItalicFace = mcRegularFace
	}
	if mcBoldItalicFace == nil {
		mcBoldItalicFace = mcBoldFace
	}
	if mcCJKFace == nil {
		mcCJKFace = mcRegularFace
	}

	// SFNT glyph inspector
	if len(assets.MinecraftRegularBytes) > 0 {
		if sfntF, err := sfnt.Parse(assets.MinecraftRegularBytes); err == nil {
			mcSfntFont = sfntF
		}
	}
}

// RenderImage generates the raw *image.RGBA canvas based on data.Style.
func RenderImage(data *WidgetData) (*image.RGBA, error) {
	if data == nil {
		data = &WidgetData{
			Style:       "default",
			Host:        "localhost",
			Port:        25565,
			Edition:     "Java Edition",
			Dark:        true,
			Rounded:     true,
			Transparent: false,
			HideIcon:    false,
			HideAddress: false,
		}
	}

	switch strings.ToLower(strings.TrimSpace(data.Style)) {
	case "minecraft", "classic", "mc", "game":
		return RenderMinecraftImage(data)
	default:
		return RenderDefaultImage(data)
	}
}

// Render generates a standard PNG image based on data.Style.
func Render(data *WidgetData) ([]byte, error) {
	img, err := RenderImage(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderWidget is a convenience wrapper for Render.
func RenderWidget(data *WidgetData) ([]byte, error) {
	return Render(data)
}

// Render on WidgetData allows calling data.Render().
func (d *WidgetData) Render() ([]byte, error) {
	return Render(d)
}
// --- Shared Graphics Primitives ---

// drawHLine draws a horizontal line from x1 to x2 at y.
func drawHLine(dst *image.RGBA, x1, x2, y int, col color.RGBA) {
	if y < 0 || y >= dst.Rect.Dy() {
		return
	}
	for x := x1; x <= x2; x++ {
		if x >= 0 && x < dst.Rect.Dx() {
			dst.Set(x, y, col)
		}
	}
}

// drawRectOutline draws a 1px rectangle outline.
func drawRectOutline(dst *image.RGBA, x, y, w, h int, col color.RGBA) {
	for i := x; i < x+w; i++ {
		if i >= 0 && i < dst.Rect.Dx() {
			if y >= 0 && y < dst.Rect.Dy() {
				dst.Set(i, y, col)
			}
			if y+h-1 >= 0 && y+h-1 < dst.Rect.Dy() {
				dst.Set(i, y+h-1, col)
			}
		}
	}
	for j := y; j < y+h; j++ {
		if j >= 0 && j < dst.Rect.Dy() {
			if x >= 0 && x < dst.Rect.Dx() {
				dst.Set(x, j, col)
			}
			if x+w-1 >= 0 && x+w-1 < dst.Rect.Dx() {
				dst.Set(x+w-1, j, col)
			}
		}
	}
}

// drawCircle draws a filled circle with center (cx, cy) and radius r.
func drawCircle(dst *image.RGBA, cx, cy, r int, col color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				x := cx + dx
				y := cy + dy
				if x >= 0 && x < dst.Rect.Dx() && y >= 0 && y < dst.Rect.Dy() {
					dst.Set(x, y, col)
				}
			}
		}
	}
}

// drawRoundedBox fills a rectangle with optional rounded corners.
func drawRoundedBox(dst *image.RGBA, x, y, w, h, radius int, col color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if radius > 0 {
				dx := 0
				if i < x+radius {
					dx = x + radius - i
				} else if i >= x+w-radius {
					dx = i - (x + w - radius - 1)
				}
				dy := 0
				if j < y+radius {
					dy = y + radius - j
				} else if j >= y+h-radius {
					dy = j - (y + h - radius - 1)
				}
				if dx*dx+dy*dy > radius*radius {
					continue
				}
			}
			dst.Set(i, j, col)
		}
	}
}

// parseHexColor parses a hex string like "#55FF55" into color.RGBA.
func parseHexColor(hexStr string, def color.RGBA) color.RGBA {
	hexStr = strings.TrimPrefix(hexStr, "#")
	if len(hexStr) != 6 {
		return def
	}
	rgb, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return def
	}
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8((rgb >> 8) & 0xFF),
		B: uint8(rgb & 0xFF),
		A: 255,
	}
}
