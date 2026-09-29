package widget

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"mcstatus/internal/assets"
	"mcstatus/internal/motd"
)

const (
	MCWidth  = 650
	MCHeight = 88
)

// hasMCGlyph checks if the Minecraft font has a glyph for the given rune.
func hasMCGlyph(r rune) bool {
	if mcSfntFont == nil {
		return r < 128
	}
	var buf sfnt.Buffer
	idx, err := mcSfntFont.GlyphIndex(&buf, r)
	return err == nil && idx != 0
}

// selectMCRuneFace returns the font.Face to render rune r with bold/italic.
func selectMCRuneFace(r rune, bold, italic bool) font.Face {
	if hasMCGlyph(r) {
		if bold && italic && mcBoldItalicFace != nil {
			return mcBoldItalicFace
		}
		if bold && mcBoldFace != nil {
			return mcBoldFace
		}
		if italic && mcItalicFace != nil {
			return mcItalicFace
		}
		if mcRegularFace != nil {
			return mcRegularFace
		}
	}
	return mcCJKFace
}

// mcShadowColor computes the Minecraft drop shadow color (brightness / 4).
func mcShadowColor(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: c.R / 4,
		G: c.G / 4,
		B: c.B / 4,
		A: c.A,
	}
}

// measureMCRune returns advance width of rune r in pixels.
func measureMCRune(r rune, bold, italic bool) int {
	face := selectMCRuneFace(r, bold, italic)
	adv, ok := face.GlyphAdvance(r)
	if !ok {
		return 8
	}
	return adv.Ceil()
}

// measureMCText measures width of plain string with given bold/italic.
func measureMCText(text string, bold, italic bool) int {
	total := 0
	for _, r := range text {
		total += measureMCRune(r, bold, italic)
	}
	return total
}

// drawMCRune renders a rune with authentic Minecraft drop shadow.
func drawMCRune(dst *image.RGBA, x, y int, r rune, col color.RGBA, bold, italic bool) int {
	face := selectMCRuneFace(r, bold, italic)
	shadow := mcShadowColor(col)

	// Draw shadow offset by +2, +2
	dShadow := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(shadow),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x + 2), Y: fixed.I(y + 2)},
	}
	dShadow.DrawString(string(r))

	// Draw foreground text
	dText := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	dText.DrawString(string(r))

	adv, ok := face.GlyphAdvance(r)
	if !ok {
		return 8
	}
	return adv.Ceil()
}

// drawMCPlain renders plain text string with specified color and drop shadow.
func drawMCPlain(dst *image.RGBA, x, y int, text string, col color.RGBA, bold, italic bool) int {
	curX := x
	for _, r := range text {
		curX += drawMCRune(dst, curX, y, r, col, bold, italic)
	}
	return curX - x
}

// drawMCSpans renders a slice of TextSpan across a line until maxW or newline.
func drawMCSpans(dst *image.RGBA, x, y int, spans []motd.TextSpan, maxW int) int {
	curX := x
	defaultColor := color.RGBA{170, 170, 170, 255} // §7 gray default

	for _, span := range spans {
		col := defaultColor
		if span.Color != "" {
			col = parseHexColor(span.Color, defaultColor)
		}

		for _, r := range span.Text {
			if r == '\n' || r == '\r' {
				return curX - x
			}
			if curX-x >= maxW {
				return curX - x
			}
			curX += drawMCRune(dst, curX, y, r, col, span.Bold, span.Italic)
		}
	}
	return curX - x
}

// drawMCPingBars renders the authentic 5-bar Minecraft latency indicator or offline cross.
func drawMCPingBars(dst *image.RGBA, x, y int, online bool) {
	if !online {
		red := color.RGBA{255, 85, 85, 255}
		redShadow := mcShadowColor(red)

		for i := 0; i < 9; i++ {
			dst.Set(x+i+1, y+i+1, redShadow)
			dst.Set(x+8-i+1, y+i+1, redShadow)
		}
		for i := 0; i < 9; i++ {
			dst.Set(x+i, y+i, red)
			dst.Set(x+8-i, y+i, red)
		}
		return
	}

	green := color.RGBA{85, 255, 85, 255}
	greenShadow := mcShadowColor(green)
	barHeights := []int{3, 5, 7, 9, 11}

	for i, h := range barHeights {
		bx := x + i*3
		by := y + (12 - h)

		for dy := 0; dy < h; dy++ {
			dst.Set(bx+1, by+dy+1, greenShadow)
			dst.Set(bx+2, by+dy+1, greenShadow)
		}
		for dy := 0; dy < h; dy++ {
			dst.Set(bx, by+dy, green)
			dst.Set(bx+1, by+dy, green)
		}
	}
}

// RenderMinecraftImage generates the raw *image.RGBA canvas for Minecraft server list style.
func RenderMinecraftImage(data *WidgetData) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, MCWidth, MCHeight))
	var (
		bgColor     color.RGBA
		borderColor color.RGBA
	)

	if data.Dark {
		bgColor = color.RGBA{14, 14, 14, 245}
		borderColor = color.RGBA{38, 38, 38, 255}
	} else {
		bgColor = color.RGBA{45, 45, 45, 245}
		borderColor = color.RGBA{70, 70, 70, 255}
	}

	if data.Transparent {
		bgColor = color.RGBA{0, 0, 0, 0}
	}

	cardRadius := 6
	if !data.Rounded {
		cardRadius = 0
	}

	// 1. Background
	if !data.Transparent {
		drawRoundedBox(img, 0, 0, MCWidth, MCHeight, cardRadius, bgColor)
		drawRectOutline(img, 0, 0, MCWidth, MCHeight, borderColor)
	}

	// 2. 64x64 Server Icon (NearestNeighbor for pixel precision)
	iconImg := data.Icon
	if iconImg == nil {
		iconImg = assets.DefaultIcon
	}
	iconRect := image.Rect(12, 12, 76, 76)
	draw.NearestNeighbor.Scale(img, iconRect, iconImg, iconImg.Bounds(), draw.Over, nil)
	drawRectOutline(img, 11, 11, 66, 66, color.RGBA{30, 30, 30, 255})

	// 3. Top Row (y = 28 baseline)
	serverTitle := data.Host
	if (data.Edition == "Java Edition" && data.Port != 25565) || (data.Edition == "Bedrock Edition" && data.Port != 19132) {
		serverTitle = net.JoinHostPort(data.Host, strconv.Itoa(int(data.Port)))
	}
	white := color.RGBA{255, 255, 255, 255}
	drawMCPlain(img, 88, 28, serverTitle, white, true, false)

	// Top right: Ping indicator
	pingX := MCWidth - 30
	pingY := 16
	drawMCPingBars(img, pingX, pingY, data.Online)

	// Top right: Players count / Status
	gray := color.RGBA{170, 170, 170, 255}
	red := color.RGBA{255, 85, 85, 255}
	playersStr := "Offline"
	playersCol := red
	if data.Online {
		playersStr = fmt.Sprintf("%d / %d", data.PlayersOnline, data.PlayersMax)
		playersCol = gray
	}
	playersW := measureMCText(playersStr, false, false)
	drawMCPlain(img, pingX-10-playersW, 28, playersStr, playersCol, false, false)

	// 4. Middle & Bottom Row: MOTD Lines (y = 50, y = 70)
	motdRaw := data.MOTDRaw
	if motdRaw == "" {
		motdRaw = data.MOTD
	}
	if !data.Online {
		motdRaw = "§cCan't connect to server."
	}
	if motdRaw == "" {
		motdRaw = "A Minecraft Server"
	}

	lines := strings.Split(motdRaw, "\n")
	maxTextW := MCWidth - 88 - 20

	if len(lines) > 0 {
		spans1 := motd.ParseSpans(lines[0])
		drawMCSpans(img, 88, 50, spans1, maxTextW)
	}
	if len(lines) > 1 {
		spans2 := motd.ParseSpans(lines[1])
		drawMCSpans(img, 88, 70, spans2, maxTextW)
	}
	return img, nil
}

// RenderMinecraft generates a 650x88 Minecraft Server List entry PNG.
func RenderMinecraft(data *WidgetData) ([]byte, error) {
	img, err := RenderMinecraftImage(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
