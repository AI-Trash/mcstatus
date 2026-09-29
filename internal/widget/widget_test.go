package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"mcstatus/internal/assets"
)

func TestRenderDefault(t *testing.T) {
	data := &WidgetData{
		Online:        true,
		Host:          "play.example.com",
		Port:          25565,
		Edition:       "Java Edition",
		Version:       "1.20.4",
		PlayersOnline: 42,
		PlayersMax:    100,
		MOTD:          "Welcome to our server!\nEnjoy your stay!",
		Dark:          true,
		Rounded:       true,
		Transparent:   false,
	}

	buf, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(buf) == 0 {
		t.Fatal("Render returned empty bytes")
	}

	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("Failed to decode rendered PNG: %v", err)
	}

	if bounds := img.Bounds(); bounds.Dx() != BannerWidth || bounds.Dy() != BannerHeight {
		t.Fatalf("Unexpected image dimensions: %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), BannerWidth, BannerHeight)
	}
}

func TestRenderOffline(t *testing.T) {
	data := &WidgetData{
		Online:      false,
		Host:        "offline.example.com",
		Port:        25565,
		Edition:     "Java Edition",
		Dark:        true,
		Rounded:     true,
		Transparent: false,
	}

	buf, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("Failed to decode rendered PNG: %v", err)
	}

	if bounds := img.Bounds(); bounds.Dx() != BannerWidth || bounds.Dy() != BannerHeight {
		t.Fatalf("Unexpected image dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderLightAndNonRounded(t *testing.T) {
	data := &WidgetData{
		Online:        true,
		Host:          "bedrock.example.com",
		Port:          19132,
		Edition:       "Bedrock Edition",
		Version:       "1.20.50",
		PlayersOnline: 5,
		PlayersMax:    20,
		MOTD:          "A Bedrock server",
		Dark:          false,
		Rounded:       false,
		Transparent:   false,
	}

	buf, err := data.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("Failed to decode rendered PNG: %v", err)
	}

	// Verify top-left pixel is border color, not transparent (since not rounded)
	r, g, b, a := img.At(0, 0).RGBA()
	if a == 0 {
		t.Fatalf("Expected non-transparent corner for non-rounded widget, got a=%d (%d,%d,%d)", a, r, g, b)
	}
}

func TestRenderTransparent(t *testing.T) {
	data := &WidgetData{
		Online:      true,
		Host:        "transparent.example.com",
		Port:        25565,
		Dark:        true,
		Rounded:     true,
		Transparent: true,
	}

	buf, err := RenderWidget(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("Failed to decode rendered PNG: %v", err)
	}

	// In transparent mode, background pixel at (10, 10) should be transparent (alpha 0)
	_, _, _, a := img.At(10, 10).RGBA()
	if a != 0 {
		t.Fatalf("Expected transparent pixel at (10, 10), got alpha=%d", a)
	}
}

func TestRenderCustomIcon(t *testing.T) {
	// Create a 64x64 solid green custom icon
	customIcon := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			customIcon.Set(x, y, color.RGBA{R: 0, G: 200, B: 0, A: 255})
		}
	}

	data := &WidgetData{
		Online:        true,
		Host:          "custom.example.com",
		Port:          25566,
		Edition:       "Java Edition",
		Icon:          customIcon,
		Dark:          true,
		Rounded:       true,
		Transparent:   false,
		PlayersOnline: 10,
		PlayersMax:    100,
	}

	buf, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(buf) == 0 {
		t.Fatal("Render returned empty bytes")
	}
}

func TestRenderNilData(t *testing.T) {
	buf, err := Render(nil)
	if err != nil {
		t.Fatalf("Render(nil) failed: %v", err)
	}
	if len(buf) == 0 {
		t.Fatal("Render returned empty bytes")
	}
}

func TestRenderDefaultIconFallback(t *testing.T) {
	data := &WidgetData{
		Online:  true,
		Host:    "demo.example.com",
		Port:    25565,
		Icon:    nil, // explicitly nil, should use assets.DefaultIcon
		Dark:    true,
		Rounded: true,
	}

	buf, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(buf) == 0 {
		t.Fatal("Render returned empty bytes")
	}

	if assets.DefaultIcon == nil {
		t.Fatal("assets.DefaultIcon is nil")
	}
}

func TestRenderMinecraftStyle(t *testing.T) {
	data := &WidgetData{
		Style:         "minecraft",
		Online:        true,
		Host:          "mc.hypixel.net",
		Port:          25565,
		Edition:       "Java Edition",
		Version:       "1.21.4",
		PlayersOnline: 35820,
		PlayersMax:    100000,
		MOTDRaw:       "§6§lHYPIXEL NETWORK §7[1.8-1.21]\n§a§lNEW UPDATE! §eSkyBlock & BedWars 欢迎游玩",
		Dark:          true,
		Rounded:       true,
	}

	buf, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("Failed to decode rendered PNG: %v", err)
	}

	if bounds := img.Bounds(); bounds.Dx() != MCWidth || bounds.Dy() != MCHeight {
		t.Fatalf("Unexpected image dimensions: %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), MCWidth, MCHeight)
	}
}

func TestRenderMinecraftOffline(t *testing.T) {
	data := &WidgetData{
		Style:   "minecraft",
		Online:  false,
		Host:    "offline.server.net",
		Port:    25565,
		Edition: "Java Edition",
	}

	buf, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("Failed to decode rendered PNG: %v", err)
	}

	if bounds := img.Bounds(); bounds.Dx() != MCWidth || bounds.Dy() != MCHeight {
		t.Fatalf("Unexpected image dimensions: %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), MCWidth, MCHeight)
	}
}

func TestRenderMinecraftAdaptiveLayout(t *testing.T) {
	base := WidgetData{
		Style:         "minecraft",
		Online:        true,
		Host:          "pure.mc.asyncraft.club",
		Port:          25565,
		PlayersOnline: 1,
		PlayersMax:    500,
		MOTDRaw:       "Line 1\nLine 2",
	}

	// 1. Hide Title via Title = "" (Canvas is 650x76)
	d1 := base
	emptyTitle := ""
	d1.Title = &emptyTitle
	buf1, err := Render(&d1)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	img1, _ := png.Decode(bytes.NewReader(buf1))
	if img1.Bounds().Dx() != MCWidth || img1.Bounds().Dy() != 76 {
		t.Fatalf("HideTitle dimension want 650x76, got %dx%d", img1.Bounds().Dx(), img1.Bounds().Dy())
	}

	// 2. Hide Icon (Canvas width is trimmed by 76px to 574, height is 80)
	d2 := base
	d2.HideIcon = true
	buf2, err := Render(&d2)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	img2, _ := png.Decode(bytes.NewReader(buf2))
	if img2.Bounds().Dx() != 574 || img2.Bounds().Dy() != 80 {
		t.Fatalf("HideIcon dimension want 574x80, got %dx%d", img2.Bounds().Dx(), img2.Bounds().Dy())
	}

	// 3. Hide Both (Canvas width is 574, height is trimmed to 52)
	d3 := base
	d3.Title = &emptyTitle
	d3.HideIcon = true
	buf3, err := Render(&d3)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	img3, _ := png.Decode(bytes.NewReader(buf3))
	if img3.Bounds().Dx() != 574 || img3.Bounds().Dy() != 52 {
		t.Fatalf("HideBoth dimension want 574x52, got %dx%d", img3.Bounds().Dx(), img3.Bounds().Dy())
	}

	// 4. Custom Title text
	d4 := base
	customTitle := "My Custom Server"
	d4.Title = &customTitle
	buf4, err := Render(&d4)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	img4, _ := png.Decode(bytes.NewReader(buf4))
	if img4.Bounds().Dx() != MCWidth || img4.Bounds().Dy() != MCHeight {
		t.Fatalf("CustomTitle dimension want 650x88, got %dx%d", img4.Bounds().Dx(), img4.Bounds().Dy())
	}
}
