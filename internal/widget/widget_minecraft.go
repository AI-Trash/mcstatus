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

// fontHasGlyph checks if the given sfnt.Font contains a glyph for rune r.
func fontHasGlyph(f *sfnt.Font, r rune) bool {
	if f == nil {
		return false
	}
	var buf sfnt.Buffer
	idx, err := f.GlyphIndex(&buf, r)
	return err == nil && idx != 0
}

// selectMCRuneFace returns the font.Face and yOffset for rune r with bold/italic.
// If the rune does not exist in any font, it returns ok=false to prevent rendering tofu boxes.
func selectMCRuneFace(r rune, bold, italic bool) (face font.Face, yOff int, ok bool) {
	// 1. Minecraft OTC font (covers ASCII 0..127 + extended Minecraft glyphs with perfect 8px grid alignment)
	if fontHasGlyph(assets.MinecraftRegular, r) {
		if bold && italic && mcBoldItalicFace != nil {
			return mcBoldItalicFace, 0, true
		}
		if bold && mcBoldFace != nil {
			return mcBoldFace, 0, true
		}
		if italic && mcItalicFace != nil {
			return mcItalicFace, 0, true
		}
		if mcRegularFace != nil {
			return mcRegularFace, 0, true
		}
	}
	// 3. GNU Unifont (covers CJK and standard Unicode blocks, shifted by -1 to align baselines)
	if fontHasGlyph(assets.Unifont, r) && mcCJKFace != nil {
		return mcCJKFace, -1, true
	}
	// 4. GNU Unifont Upper (covers emojis like 🎮, pictographs, symbols)
	if fontHasGlyph(assets.UnifontUpper, r) && mcUpperFace != nil {
		return mcUpperFace, -1, true
	}
	// 5. Zpix fallback (covers game symbols, pixel art brackets, and missing CJK)
	if fontHasGlyph(assets.DefaultFont, r) && mcZpixFace != nil {
		return mcZpixFace, -1, true
	}
	return nil, 0, false
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
	face, _, ok := selectMCRuneFace(r, bold, italic)
	if !ok || face == nil {
		return 8
	}
	adv, advOk := face.GlyphAdvance(r)
	width := 8
	if advOk {
		width = adv.Ceil()
	}
	isOTC := fontHasGlyph(assets.MinecraftRegular, r)
	if bold && !isOTC {
		width += 2
	}
	return width
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
// If a CJK rune has bold enabled, it simulates Minecraft's 1px horizontal offset rendering.
func drawMCRune(dst *image.RGBA, x, y int, r rune, col color.RGBA, bold, italic bool) int {
	face, yOff, ok := selectMCRuneFace(r, bold, italic)
	if !ok || face == nil {
		return 8
	}

	drawY := y + yOff
	shadow := mcShadowColor(col)

	// Draw shadow offset by +2, +2
	dShadow := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(shadow),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x + 2), Y: fixed.I(drawY + 2)},
	}
	dShadow.DrawString(string(r))

	isOTC := fontHasGlyph(assets.MinecraftRegular, r)
	if bold && !isOTC {
		dShadowBold := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(shadow),
			Face: face,
			Dot:  fixed.Point26_6{X: fixed.I(x + 4), Y: fixed.I(drawY + 2)},
		}
		dShadowBold.DrawString(string(r))
	}

	// Draw foreground text
	dText := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(drawY)},
	}
	dText.DrawString(string(r))

	if bold && !isOTC {
		dTextBold := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(col),
			Face: face,
			Dot:  fixed.Point26_6{X: fixed.I(x + 2), Y: fixed.I(drawY)},
		}
		dTextBold.DrawString(string(r))
	}

	adv, advOk := face.GlyphAdvance(r)
	width := 8
	if advOk {
		width = adv.Ceil()
	}
	if bold && !isOTC {
		width += 2
	}
	return width
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
// It dynamically adapts canvas height and text positioning when ShowIcon or ShowAddress are toggled.
func RenderMinecraftImage(data *WidgetData) (*image.RGBA, error) {
	showIcon := true
	hasTitle := true
	titleText := "localhost"

	if data != nil {
		showIcon = !data.HideIcon
		titleText = data.Host
		if (data.Edition == "Java Edition" && data.Port != 25565) || (data.Edition == "Bedrock Edition" && data.Port != 19132) {
			titleText = net.JoinHostPort(data.Host, strconv.Itoa(int(data.Port)))
		}
		if data.Title != nil {
			if *data.Title == "" {
				hasTitle = false
			} else {
				hasTitle = true
				titleText = *data.Title
			}
		}
	}

	// 1. Self-adaptive canvas width and height
	canvasW := MCWidth // 650
	if !showIcon {
		canvasW = MCWidth - 76 // 574 (completely trim out 64px icon + padding)
	}

	canvasH := MCHeight // default 88
	if hasTitle {
		if !showIcon {
			canvasH = 80 // icon gone, text lines fit in 80px
		}
	} else {
		if showIcon {
			canvasH = 76 // fits 64px icon with 6px padding
		} else {
			canvasH = 52 // ultra-compact: both icon and title gone, 2 lines of text
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
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

	// 2. Background and border
	if !data.Transparent {
		drawRoundedBox(img, 0, 0, canvasW, canvasH, cardRadius, bgColor)
		drawRoundedOutline(img, 0, 0, canvasW, canvasH, cardRadius, borderColor)
	}

	// 3. Self-adaptive startX and icon rendering
	startX := 14
	if showIcon {
		startX = 88
		iconY := 12
		if !hasTitle {
			iconY = 6
		}
		iconImg := data.Icon
		if iconImg == nil {
			iconImg = assets.DefaultIcon
		}
		iconRect := image.Rect(12, iconY, 76, iconY+64)
		draw.NearestNeighbor.Scale(img, iconRect, iconImg, iconImg.Bounds(), draw.Over, nil)
		drawRectOutline(img, 11, iconY-1, 66, 66, color.RGBA{30, 30, 30, 255})
	}

	// 4. Ping indicator & Players count on the right
	pingX := canvasW - 28
	pingY := 16
	playersY := 28
	if !hasTitle {
		pingY = 12
		playersY = 24
	}
	drawMCPingBars(img, pingX, pingY, data.Online)

	gray := color.RGBA{170, 170, 170, 255}
	red := color.RGBA{255, 85, 85, 255}
	playersStr := "Offline"
	playersCol := red
	if data.Online {
		playersStr = fmt.Sprintf("%d / %d", data.PlayersOnline, data.PlayersMax)
		playersCol = gray
	}
	playersW := measureMCText(playersStr, false, false)
	playersX := pingX - 10 - playersW
	drawMCPlain(img, playersX, playersY, playersStr, playersCol, false, false)

	// 5. Server Title (only if hasTitle is true)
	if hasTitle {
		white := color.RGBA{255, 255, 255, 255}
		drawMCPlain(img, startX, 28, titleText, white, false, false)
	}
	// 6. MOTD Lines with adaptive positioning and text width clamping
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
	motdY1 := 50
	motdY2 := 70
	maxLine1W := canvasW - startX - 16
	maxLine2W := canvasW - startX - 16

	if !hasTitle {
		// When title is hidden, MOTD line 1 moves up to line 1!
		motdY1 = 24
		motdY2 = 44
		// Line 1 stops before players count to prevent overlap
		maxLine1W = playersX - startX - 10
	}
	if len(lines) > 0 {
		spans1 := motd.ParseSpans(lines[0])
		drawMCSpans(img, startX, motdY1, spans1, maxLine1W)
	}
	if len(lines) > 1 {
		spans2 := motd.ParseSpans(lines[1])
		drawMCSpans(img, startX, motdY2, spans2, maxLine2W)
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
