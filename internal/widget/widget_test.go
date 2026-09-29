package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	gavif "github.com/gen2brain/gav1d/avif"
	"github.com/gen2brain/jpegxl"
	"github.com/gen2brain/webp"
	_ "golang.org/x/image/font/sfnt"

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
func TestRenderDefaultAdaptiveLayout(t *testing.T) {
	emptyTitle := ""
	base := WidgetData{
		Online:        true,
		Host:          "play.example.com",
		Port:          25565,
		Edition:       "Java Edition",
		Version:       "1.20.4",
		PlayersOnline: 10,
		PlayersMax:    100,
		MOTD:          "Welcome!",
		Dark:          true,
	}

	// 1. Hide Icon (760 x 240)
	d1 := base
	d1.HideIcon = true
	buf1, _ := Render(&d1)
	img1, _ := png.Decode(bytes.NewReader(buf1))
	if img1.Bounds().Dx() != 760 || img1.Bounds().Dy() != 240 {
		t.Fatalf("Default HideIcon dimension want 760x240, got %dx%d", img1.Bounds().Dx(), img1.Bounds().Dy())
	}

	// 2. Hide Title (860 x 208)
	d2 := base
	d2.Title = &emptyTitle
	buf2, _ := Render(&d2)
	img2, _ := png.Decode(bytes.NewReader(buf2))
	if img2.Bounds().Dx() != 860 || img2.Bounds().Dy() != 208 {
		t.Fatalf("Default HideTitle dimension want 860x208, got %dx%d", img2.Bounds().Dx(), img2.Bounds().Dy())
	}

	// 3. Hide Both (760 x 208)
	d3 := base
	d3.HideIcon = true
	d3.Title = &emptyTitle
	buf3, _ := Render(&d3)
	img3, _ := png.Decode(bytes.NewReader(buf3))
	if img3.Bounds().Dx() != 760 || img3.Bounds().Dy() != 208 {
		t.Fatalf("Default HideBoth dimension want 760x208, got %dx%d", img3.Bounds().Dx(), img3.Bounds().Dy())
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
func TestGenerateReviewImage(t *testing.T) {
	data := &WidgetData{
		Online:        true,
		Host:          "games.mc.asyncraft.club",
		Port:          25565,
		Edition:       "Java Edition",
		Version:       "1.20.4",
		PlayersOnline: 0,
		PlayersMax:    500,
		MOTD:          "Asyncraft服务器\n -> 🎮 小游戏 🎮",
		MOTDRaw:       "§#55ffffA§#74ffffs§#93ffffy§#b2ffffn§#d1ffffc§#f0ffffra§#d1fffff§#b2fffft§#93ffff服§#74ffff务§#55ffff器\n§e -> 🎮 小游戏 🎮",
		Dark:          true,
		Rounded:       false,
		Transparent:   false,
	}

	buf, err := RenderMinecraft(data)
	if err != nil {
		t.Fatal(err)
	}

	outPath := "C:/Users/logen/AppData/Local/Temp/review_widget.png"
	if err := os.WriteFile(outPath, buf, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("Successfully wrote %s", outPath)
}
func TestBenchmarkRealWidgetSizes(t *testing.T) {
	data := &WidgetData{
		Online:        true,
		Host:          "games.mc.asyncraft.club",
		Port:          25565,
		Edition:       "Java Edition",
		Version:       "1.20.4",
		PlayersOnline: 42,
		PlayersMax:    500,
		MOTD:          "Asyncraft服务器\n -> 🎮 小游戏 🎮",
		MOTDRaw:       "§#55ffffA§#74ffffs§#93ffffy§#b2ffffn§#d1ffffc§#f0ffffra§#d1fffff§#b2fffft§#93ffff服§#74ffff务§#55ffff器\n§e -> 🎮 小游戏 🎮",
		Dark:          true,
		Rounded:       false,
		Transparent:   false,
	}

	cases := []struct {
		style string
	}{
		{"minecraft"},
		{"default"},
	}

	tempDir := os.Getenv("TEMP")

	for _, tc := range cases {
		d := *data
		d.Style = tc.style
		rawImg, err := RenderImage(&d)
		if err != nil {
			t.Fatalf("RenderImage(%s) failed: %v", tc.style, err)
		}

		t.Logf("=================================================================")
		t.Logf(" REAL WIDGET BENCHMARK: style=%s (bounds=%v)", tc.style, rawImg.Bounds())
		t.Logf("=================================================================")

		// 1. Lossless PNG
		t0 := time.Now()
		var pngBuf bytes.Buffer
		_ = png.Encode(&pngBuf, rawImg)
		durPNG := time.Since(t0)
		pngSize := pngBuf.Len()
		_ = os.WriteFile(filepath.Join(tempDir, "bench_"+tc.style+".png"), pngBuf.Bytes(), 0644)
		t.Logf(" [Lossless PNG]        %6d bytes | 100.0%% (baseline) | %v", pngSize, durPNG)

		// 2. Lossless WebP (Default Method 4)
		t0 = time.Now()
		var webpBuf4 bytes.Buffer
		_ = webp.Encode(&webpBuf4, rawImg, webp.Options{Lossless: true, Method: 4})
		durWebP4 := time.Since(t0)
		t.Logf(" [Lossless WebP m=4]   %6d bytes | %5.1f%% vs PNG    | %v", webpBuf4.Len(), float64(webpBuf4.Len())/float64(pngSize)*100, durWebP4)

		// 3. Lossless WebP (Max Method 6)
		t0 = time.Now()
		var webpBuf6 bytes.Buffer
		_ = webp.Encode(&webpBuf6, rawImg, webp.Options{Lossless: true, Method: 6})
		durWebP6 := time.Since(t0)
		_ = os.WriteFile(filepath.Join(tempDir, "bench_"+tc.style+".webp"), webpBuf6.Bytes(), 0644)
		t.Logf(" [Lossless WebP m=6]   %6d bytes | %5.1f%% vs PNG    | %v", webpBuf6.Len(), float64(webpBuf6.Len())/float64(pngSize)*100, durWebP6)

		// 4. Lossless JXL (Effort 7)
		t0 = time.Now()
		var jxlBuf7 bytes.Buffer
		_ = jpegxl.Encode(&jxlBuf7, rawImg, jpegxl.Options{Lossless: true, Effort: 7})
		durJXL7 := time.Since(t0)
		t.Logf(" [Lossless JXL e=7]    %6d bytes | %5.1f%% vs PNG    | %v", jxlBuf7.Len(), float64(jxlBuf7.Len())/float64(pngSize)*100, durJXL7)

		// 5. Lossless JXL (Effort 9)
		t0 = time.Now()
		var jxlBuf9 bytes.Buffer
		_ = jpegxl.Encode(&jxlBuf9, rawImg, jpegxl.Options{Lossless: true, Effort: 9})
		durJXL9 := time.Since(t0)
		_ = os.WriteFile(filepath.Join(tempDir, "bench_"+tc.style+".jxl"), jxlBuf9.Bytes(), 0644)
		t.Logf(" [Lossless JXL e=9]    %6d bytes | %5.1f%% vs PNG    | %v", jxlBuf9.Len(), float64(jxlBuf9.Len())/float64(pngSize)*100, durJXL9)

		// 6. Lossless AVIF (Speed 6)
		t0 = time.Now()
		var avifBuf bytes.Buffer
		_ = gavif.Encode(&avifBuf, rawImg, gavif.EncodeOptions{Lossless: true, Speed: 6})
		durAVIF := time.Since(t0)
		_ = os.WriteFile(filepath.Join(tempDir, "bench_"+tc.style+".avif"), avifBuf.Bytes(), 0644)
		t.Logf(" [Lossless AVIF s=6]   %6d bytes | %5.1f%% vs PNG    | %v", avifBuf.Len(), float64(avifBuf.Len())/float64(pngSize)*100, durAVIF)
	}
}
