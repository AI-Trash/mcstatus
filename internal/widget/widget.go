package widget

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"mcstatus/internal/assets"
)

const (
	BannerWidth  = 860
	BannerHeight = 240
	CornerRadius = 16
	IconX        = 32
	IconY        = 40
	IconSize     = 80
)

var (
	titleFace  font.Face
	normalFace font.Face
)

func init() {
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
}

func measureText(text string) int {
	return (&font.Drawer{Face: normalFace}).MeasureString(text).Ceil()
}

func drawTitle(dst *image.RGBA, x, y int, text string, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: titleFace,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + 16)},
	}
	d.DrawString(text)
}

func drawText(dst *image.RGBA, x, y int, text string, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: normalFace,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + 11)},
	}
	d.DrawString(text)
}

// WidgetData holds all parameters needed to render a server status widget banner.
type WidgetData struct {
	Online        bool
	Host          string
	Port          uint16
	Edition       string // ("Java Edition" or "Bedrock Edition")
	Version       string
	PlayersOnline int
	PlayersMax    int
	MOTD          string // (clean text)
	Icon          image.Image // (if nil, uses assets.DefaultIcon)
	Dark          bool // (default true)
	Rounded       bool // (default true)
	Transparent   bool // (default false)
}

// Render generates an 860x240 PNG image banner from the provided WidgetData.
func Render(data *WidgetData) ([]byte, error) {
	if data == nil {
		data = &WidgetData{
			Dark:    true,
			Rounded: true,
		}
	}

	var (
		bgColor       color.RGBA
		borderColor   color.RGBA
		primaryText   color.RGBA
		secondaryText color.RGBA
		motdText      color.RGBA
		dividerColor  color.RGBA
		brandingCol   color.RGBA
	)

	if data.Dark {
		bgColor = color.RGBA{R: 0x16, G: 0x16, B: 0x18, A: 0xff}       // #161618
		borderColor = color.RGBA{R: 0x2e, G: 0x2e, B: 0x32, A: 0xff}   // #2e2e32
		primaryText = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}   // white
		secondaryText = color.RGBA{R: 0x8e, G: 0x8e, B: 0x93, A: 0xff} // gray
		motdText = color.RGBA{R: 0xd1, G: 0xd1, B: 0xd6, A: 0xff}      // light gray
		dividerColor = color.RGBA{R: 0x2e, G: 0x2e, B: 0x32, A: 0xff}  // dark divider
		brandingCol = color.RGBA{R: 0x63, G: 0x63, B: 0x66, A: 0xff}   // muted
	} else {
		bgColor = color.RGBA{R: 0xf8, G: 0xf9, B: 0xfa, A: 0xff}       // #f8f9fa
		borderColor = color.RGBA{R: 0xe5, G: 0xe7, B: 0xeb, A: 0xff}   // #e5e7eb
		primaryText = color.RGBA{R: 0x11, G: 0x18, B: 0x27, A: 0xff}   // dark
		secondaryText = color.RGBA{R: 0x6b, G: 0x72, B: 0x80, A: 0xff} // medium gray
		motdText = color.RGBA{R: 0x37, G: 0x41, B: 0x51, A: 0xff}      // dark gray
		dividerColor = color.RGBA{R: 0xe5, G: 0xe7, B: 0xeb, A: 0xff}  // light divider
		brandingCol = color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff}   // muted
	}

	img := image.NewRGBA(image.Rect(0, 0, BannerWidth, BannerHeight))

	if !data.Transparent {
		for y := range BannerHeight {
			for x := range BannerWidth {
				img.SetRGBA(x, y, bgColor)
			}
		}

		if data.Rounded {
			r := CornerRadius
			for y := range r {
				for x := range r {
					dx := r - x
					dy := r - y
					dist := math.Sqrt(float64(dx*dx + dy*dy))
					if dist > float64(r) {
						img.SetRGBA(x, y, color.RGBA{})
						img.SetRGBA(BannerWidth-1-x, y, color.RGBA{})
						img.SetRGBA(x, BannerHeight-1-y, color.RGBA{})
						img.SetRGBA(BannerWidth-1-x, BannerHeight-1-y, color.RGBA{})
					}
				}
			}
		}

		drawRectOutline(img, 0, 0, BannerWidth, BannerHeight, borderColor)
	}

	serverIcon := data.Icon
	if serverIcon == nil {
		serverIcon = assets.DefaultIcon
	}

	if serverIcon != nil {
		iconTargetRect := image.Rect(IconX, IconY, IconX+IconSize, IconY+IconSize)
		draw.ApproxBiLinear.Scale(img, iconTargetRect, serverIcon, serverIcon.Bounds(), draw.Over, nil)
		drawRectOutline(img, IconX-1, IconY-1, IconSize+2, IconSize+2, borderColor)
	}

	addr := data.Host
	if data.Port != 0 && data.Port != 25565 && data.Port != 19132 && !strings.Contains(data.Host, ":") {
		addr = net.JoinHostPort(data.Host, strconv.Itoa(int(data.Port)))
	}
	if len(addr) > 36 {
		addr = addr[:33] + "..."
	}
	drawTitle(img, 136, 36, addr, primaryText)

	var (
		statusDotCol  color.RGBA
		statusTextStr string
	)
	if data.Online {
		statusDotCol = color.RGBA{R: 0x22, G: 0xc5, B: 0x5e, A: 0xff} // #22c55e
		statusTextStr = "ONLINE"
	} else {
		statusDotCol = color.RGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff} // #ef4444
		statusTextStr = "OFFLINE"
	}

	const rightEdge = BannerWidth - 32
	statusTextW := measureText(statusTextStr)
	dotRadius := 5
	statusTotalW := (dotRadius * 2) + 8 + statusTextW
	statusStartX := rightEdge - statusTotalW

	drawCircle(img, statusStartX+dotRadius, 40+7, dotRadius, statusDotCol)
	drawText(img, statusStartX+(dotRadius*2)+8, 42, statusTextStr, statusDotCol)

	editionStr := data.Edition
	if editionStr == "" {
		editionStr = "Java Edition"
	}
	if data.Version != "" {
		editionStr += " - " + data.Version
	}
	if len(editionStr) > 45 {
		editionStr = editionStr[:42] + "..."
	}
	drawText(img, 136, 74, editionStr, secondaryText)

	var playerStr string
	if data.Online {
		playerStr = fmt.Sprintf("Players: %d / %d", data.PlayersOnline, data.PlayersMax)
	} else {
		playerStr = "Players: - / -"
	}
	playerW := measureText(playerStr)
	drawText(img, rightEdge-playerW, 74, playerStr, secondaryText)

	drawHLine(img, 136, rightEdge, 104, dividerColor)

	motdClean := strings.TrimSpace(data.MOTD)
	if motdClean == "" {
		if data.Online {
			motdClean = "A Minecraft Server"
		} else {
			motdClean = "Server is currently offline"
		}
	}
	lines := strings.Split(motdClean, "\n")
	line1 := ""
	line2 := ""
	if len(lines) > 0 {
		line1 = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 {
		line2 = strings.TrimSpace(lines[1])
	}


	if len(line1) > 95 {
		line1 = line1[:92] + "..."
	}
	if len(line2) > 95 {
		line2 = line2[:92] + "..."
	}

	if line1 != "" {
		drawText(img, 136, 122, line1, motdText)
	}
	if line2 != "" {
		drawText(img, 136, 144, line2, motdText)
	}

	branding := "mcstatus.io"
	brandingW := measureText(branding)
	drawText(img, rightEdge-brandingW, 206, branding, brandingCol)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode png: %w", err)
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

func drawHLine(dst *image.RGBA, x1, x2, y int, col color.RGBA) {
	if y < 0 || y >= BannerHeight {
		return
	}
	if x1 < 0 {
		x1 = 0
	}
	if x2 >= BannerWidth {
		x2 = BannerWidth - 1
	}
	for x := x1; x <= x2; x++ {
		dst.SetRGBA(x, y, col)
	}
}

func drawRectOutline(dst *image.RGBA, x, y, w, h int, col color.RGBA) {
	for cx := x; cx < x+w; cx++ {
		if cx >= 0 && cx < BannerWidth {
			if y >= 0 && y < BannerHeight {
				dst.SetRGBA(cx, y, col)
			}
			if y+h-1 >= 0 && y+h-1 < BannerHeight {
				dst.SetRGBA(cx, y+h-1, col)
			}
		}
	}
	for cy := y; cy < y+h; cy++ {
		if cy >= 0 && cy < BannerHeight {
			if x >= 0 && x < BannerWidth {
				dst.SetRGBA(x, cy, col)
			}
			if x+w-1 >= 0 && x+w-1 < BannerWidth {
				dst.SetRGBA(x+w-1, cy, col)
			}
		}
	}
}

func drawCircle(dst *image.RGBA, cx, cy, r int, col color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				x := cx + dx
				y := cy + dy
				if x >= 0 && x < BannerWidth && y >= 0 && y < BannerHeight {
					dst.SetRGBA(x, y, col)
				}
			}
		}
	}
}
