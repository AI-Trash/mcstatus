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
		bgColor = color.RGBA{R: 0x1e, G: 0x1e, B: 0x24, A: 0xff}
		borderColor = color.RGBA{R: 0x33, G: 0x33, B: 0x3e, A: 0xff}
		primaryText = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		secondaryText = color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff}
		motdText = color.RGBA{R: 0xd4, G: 0xd4, B: 0xd8, A: 0xff}
		dividerColor = color.RGBA{R: 0x33, G: 0x33, B: 0x3e, A: 0xff}
		brandingCol = color.RGBA{R: 0x71, G: 0x71, B: 0x7a, A: 0xff}
	} else {
		bgColor = color.RGBA{R: 0xf8, G: 0xf9, B: 0xfa, A: 0xff}
		borderColor = color.RGBA{R: 0xe5, G: 0xe7, B: 0xeb, A: 0xff}
		primaryText = color.RGBA{R: 0x11, G: 0x18, B: 0x27, A: 0xff}
		secondaryText = color.RGBA{R: 0x6b, G: 0x72, B: 0x80, A: 0xff}
		motdText = color.RGBA{R: 0x37, G: 0x41, B: 0x51, A: 0xff}
		dividerColor = color.RGBA{R: 0xe5, G: 0xe7, B: 0xeb, A: 0xff}
		brandingCol = color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff}
	}

	img := image.NewRGBA(image.Rect(0, 0, BannerWidth, BannerHeight))

	// Draw background and border
	if !data.Transparent {
		r := float64(CornerRadius)
		for y := range BannerHeight {
			for x := range BannerWidth {
				if !data.Rounded {
					if x == 0 || x == BannerWidth-1 || y == 0 || y == BannerHeight-1 {
						img.SetRGBA(x, y, borderColor)
					} else {
						img.SetRGBA(x, y, bgColor)
					}
					continue
				}

				// Rounded corners check
				var dx, dy float64
				isCorner := false

				if x < CornerRadius && y < CornerRadius { // top-left
					dx = float64(CornerRadius - x)
					dy = float64(CornerRadius - y)
					isCorner = true
				} else if x >= BannerWidth-CornerRadius && y < CornerRadius { // top-right
					dx = float64(x - (BannerWidth - 1 - CornerRadius))
					dy = float64(CornerRadius - y)
					isCorner = true
				} else if x < CornerRadius && y >= BannerHeight-CornerRadius { // bottom-left
					dx = float64(CornerRadius - x)
					dy = float64(y - (BannerHeight - 1 - CornerRadius))
					isCorner = true
				} else if x >= BannerWidth-CornerRadius && y >= BannerHeight-CornerRadius { // bottom-right
					dx = float64(x - (BannerWidth - 1 - CornerRadius))
					dy = float64(y - (BannerHeight - 1 - CornerRadius))
					isCorner = true
				}

				if isCorner {
					dist := math.Hypot(dx, dy)
					if dist > r {
						// Outside corner: leave transparent alpha 0
						continue
					} else if dist >= r-1.0 {
						img.SetRGBA(x, y, borderColor)
					} else {
						img.SetRGBA(x, y, bgColor)
					}
				} else {
					if x == 0 || x == BannerWidth-1 || y == 0 || y == BannerHeight-1 {
						img.SetRGBA(x, y, borderColor)
					} else {
						img.SetRGBA(x, y, bgColor)
					}
				}
			}
		}
	}

	// Draw Server Icon at (32, 40), size 80x80
	icon := data.Icon
	if icon == nil {
		icon = assets.DefaultIcon
	}
	if icon != nil {
		iconRect := image.Rect(IconX, IconY, IconX+IconSize, IconY+IconSize)
		draw.BiLinear.Scale(img, iconRect, icon, icon.Bounds(), draw.Over, nil)
		drawRectOutline(img, IconX-1, IconY-1, IconSize+2, IconSize+2, borderColor)
	}

	// Server address: Host or Host:Port
	addr := data.Host
	if addr == "" {
		addr = "127.0.0.1"
	}
	if data.Port != 0 && data.Port != 25565 && data.Port != 19132 && !strings.Contains(data.Host, ":") {
		addr = net.JoinHostPort(data.Host, strconv.Itoa(int(data.Port)))
	}
	if len(addr) > 36 {
		addr = addr[:33] + "..."
	}
	drawText2x(img, 136, 40, addr, primaryText)

	// Status Dot and ONLINE / OFFLINE text
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
	statusTextW := len(statusTextStr) * 7
	dotRadius := 5
	statusTotalW := (dotRadius * 2) + 8 + statusTextW
	statusStartX := rightEdge - statusTotalW

	drawCircle(img, statusStartX+dotRadius, 40+7, dotRadius, statusDotCol)
	drawText(img, statusStartX+(dotRadius*2)+8, 42, statusTextStr, statusDotCol)

	// Edition & Version
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

	// Player count
	var playerStr string
	if data.Online {
		playerStr = fmt.Sprintf("Players: %d / %d", data.PlayersOnline, data.PlayersMax)
	} else {
		playerStr = "Players: - / -"
	}
	playerW := len(playerStr) * 7
	drawText(img, rightEdge-playerW, 74, playerStr, secondaryText)

	// Horizontal divider line
	drawHLine(img, 136, rightEdge, 104, dividerColor)

	// MOTD (line 1 & line 2)
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

	// mcstatus.io branding at bottom right
	branding := "mcstatus.io"
	brandingW := len(branding) * 7
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

func drawText(dst *image.RGBA, x, y int, text string, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + 11)},
	}
	d.DrawString(text)
}

func drawText2x(dst *image.RGBA, startX, startY int, text string, col color.Color) {
	if text == "" {
		return
	}
	w := len(text) * 7
	h := 13
	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: 0, Y: fixed.I(11)},
	}
	d.DrawString(text)

	cR, cG, cB, cA := col.RGBA()
	colRGBA := color.RGBA{R: uint8(cR >> 8), G: uint8(cG >> 8), B: uint8(cB >> 8), A: uint8(cA >> 8)}
	for y := range h {
		for x := range w {
			_, _, _, a := tmp.At(x, y).RGBA()
			if a > 0 {
				dx := startX + x*2
				dy := startY + y*2
				for sy := range 2 {
					for sx := range 2 {
						if dx+sx < BannerWidth && dy+sy < BannerHeight && dx+sx >= 0 && dy+sy >= 0 {
							dst.SetRGBA(dx+sx, dy+sy, colRGBA)
						}
					}
				}
			}
		}
	}
}
