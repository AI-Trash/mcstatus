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
	"golang.org/x/image/math/fixed"

	"mcstatus/internal/assets"
)

const (
	BannerWidth  = 860
	BannerHeight = 240
	IconX        = 32
	IconY        = 40
	IconSize     = 80
)

func drawTitle(dst *image.RGBA, x, y int, text string, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: titleFace,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + 16)},
	}
	d.DrawString(text)
}

func drawNormalText(dst *image.RGBA, x, y int, text string, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: normalFace,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + 11)},
	}
	d.DrawString(text)
}

// RenderDefaultImage generates the raw *image.RGBA banner for default style.
func RenderDefaultImage(data *WidgetData) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, BannerWidth, BannerHeight))
	var (
		bgColor      color.RGBA
		cardBgColor  color.RGBA
		borderColor  color.RGBA
		primaryText  color.RGBA
		secText     color.RGBA
	)

	if data.Dark {
		bgColor = color.RGBA{18, 18, 24, 255}
		cardBgColor = color.RGBA{28, 28, 38, 255}
		borderColor = color.RGBA{45, 45, 60, 255}
		primaryText = color.RGBA{240, 240, 245, 255}
		secText = color.RGBA{140, 140, 160, 255}
	} else {
		bgColor = color.RGBA{248, 249, 250, 255}
		cardBgColor = color.RGBA{255, 255, 255, 255}
		borderColor = color.RGBA{220, 224, 230, 255}
		primaryText = color.RGBA{20, 25, 35, 255}
		secText = color.RGBA{100, 110, 125, 255}
	}

	if data.Transparent {
		bgColor = color.RGBA{0, 0, 0, 0}
	}

	// 1. Fill base canvas
	for y := 0; y < BannerHeight; y++ {
		for x := 0; x < BannerWidth; x++ {
			if !data.Transparent {
				img.Set(x, y, bgColor)
			}
		}
	}

	// 2. Draw card background
	cardX, cardY, cardW, cardH := 12, 12, BannerWidth-24, BannerHeight-24
	cardRadius := 12
	if !data.Rounded {
		cardRadius = 0
	}
	drawRoundedBox(img, cardX, cardY, cardW, cardH, cardRadius, cardBgColor)
	drawRectOutline(img, cardX, cardY, cardW, cardH, borderColor)

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

	contentX := 40
	if showIcon {
		contentX = 136
		iconImg := data.Icon
		if iconImg == nil {
			iconImg = assets.DefaultIcon
		}
		iconRect := image.Rect(IconX, IconY, IconX+IconSize, IconY+IconSize)
		draw.BiLinear.Scale(img, iconRect, iconImg, iconImg.Bounds(), draw.Over, nil)
		drawRectOutline(img, IconX-1, IconY-1, IconSize+2, IconSize+2, borderColor)
	}

	// 4. Server Title
	badgeY := 76
	lineY := 114
	motdY1 := 126
	motdY2 := 146

	if hasTitle {
		drawTitle(img, contentX, 44, titleText, primaryText)
	} else {
		// When address is hidden, shift badge and MOTD up!
		badgeY = 44
		lineY = 82
		motdY1 = 100
		motdY2 = 124
	}
	// 5. Status Badge
	badgeX, badgeW, badgeH := contentX, 108, 24
	badgeRadius := 4
	if !data.Rounded {
		badgeRadius = 0
	}

	var (
		badgeBg     color.RGBA
		badgeBorder color.RGBA
		badgeText   color.RGBA
		statusLabel string
	)

	if data.Online {
		statusLabel = "ONLINE"
		if data.Dark {
			badgeBg = color.RGBA{20, 48, 32, 255}
			badgeBorder = color.RGBA{38, 115, 70, 255}
			badgeText = color.RGBA{46, 204, 113, 255}
		} else {
			badgeBg = color.RGBA{236, 253, 245, 255}
			badgeBorder = color.RGBA{167, 243, 208, 255}
			badgeText = color.RGBA{5, 150, 105, 255}
		}
	} else {
		statusLabel = "OFFLINE"
		if data.Dark {
			badgeBg = color.RGBA{48, 20, 24, 255}
			badgeBorder = color.RGBA{120, 40, 48, 255}
			badgeText = color.RGBA{239, 68, 68, 255}
		} else {
			badgeBg = color.RGBA{254, 242, 242, 255}
			badgeBorder = color.RGBA{254, 202, 202, 255}
			badgeText = color.RGBA{220, 38, 38, 255}
		}
	}

	drawRoundedBox(img, badgeX, badgeY, badgeW, badgeH, badgeRadius, badgeBg)
	drawRectOutline(img, badgeX, badgeY, badgeW, badgeH, badgeBorder)
	drawCircle(img, badgeX+14, badgeY+12, 3, badgeText)
	drawNormalText(img, badgeX+24, badgeY+4, statusLabel, badgeText)
	// 6. Edition & Version
	edX := badgeX + badgeW + 12
	edLabel := data.Edition
	if data.Version != "" {
		edLabel += " " + data.Version
	}
	drawNormalText(img, edX, badgeY+5, edLabel, secText)

	drawHLine(img, contentX, BannerWidth-48, lineY, borderColor)

	// 7. MOTD
	motdText := data.MOTD
	if !data.Online {
		motdText = "Server is currently offline or unreachable."
	}
	if motdText == "" {
		motdText = "A Minecraft Server"
	}
	motdLines := strings.Split(motdText, "\n")
	if len(motdLines) > 0 {
		drawNormalText(img, contentX, motdY1, strings.TrimSpace(motdLines[0]), primaryText)
	}
	if len(motdLines) > 1 {
		drawNormalText(img, contentX, motdY2, strings.TrimSpace(motdLines[1]), secText)
	}

	// 8. Stats Boxes (PLAYERS, PROTOCOL, PING)
	boxW, boxH, boxY := 200, 46, 170

	box1X := contentX
	drawRectOutline(img, box1X, boxY, boxW, boxH, borderColor)
	drawNormalText(img, box1X+14, boxY+6, "PLAYERS", secText)
	playersVal := "0 / 0"
	if data.Online {
		playersVal = fmt.Sprintf("%d / %d", data.PlayersOnline, data.PlayersMax)
	}
	drawNormalText(img, box1X+14, boxY+24, playersVal, primaryText)

	box2X := box1X + boxW + 16
	drawRectOutline(img, box2X, boxY, boxW, boxH, borderColor)
	drawNormalText(img, box2X+14, boxY+6, "PROTOCOL", secText)
	protoVal := "Unknown"
	if data.Online && data.Version != "" {
		protoVal = data.Version
	}
	drawNormalText(img, box2X+14, boxY+24, protoVal, primaryText)

	box3X := box2X + boxW + 16
	drawRectOutline(img, box3X, boxY, boxW, boxH, borderColor)
	drawNormalText(img, box3X+14, boxY+6, "PING", secText)
	pingVal := "N/A"
	if data.Online {
		pingVal = "< 50 ms"
	}
	drawNormalText(img, box3X+14, boxY+24, pingVal, badgeText)
	return img, nil
}

// RenderDefault generates an 860x240 PNG image banner from the provided WidgetData.
func RenderDefault(data *WidgetData) ([]byte, error) {
	img, err := RenderDefaultImage(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
