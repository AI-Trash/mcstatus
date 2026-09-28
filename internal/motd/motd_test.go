package motd

import (
	"strings"
	"testing"
)

func TestFormatHexChatComponent(t *testing.T) {
	comp := map[string]any{
		"text": "",
		"extra": []any{
			map[string]any{
				"text":  "A",
				"color": "aqua",
			},
			map[string]any{
				"text":  "syncraft",
				"color": "#74FFFF",
			},
		},
	}

	result := Format(comp)
	if result.Clean != "Asyncraft" {
		t.Fatalf("expected Clean 'Asyncraft', got '%s'", result.Clean)
	}

	if !strings.Contains(result.HTML, `style="color: #55ffff;"`) {
		t.Fatalf("expected HTML to contain aqua color, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, `style="color: #74ffff;"`) {
		t.Fatalf("expected HTML to contain #74ffff hex color, got: %s", result.HTML)
	}
}

func TestFormatLegacyBungeeHex(t *testing.T) {
	raw := "§x§7§4§f§f§f§fHello World"
	result := Format(raw)

	if result.Clean != "Hello World" {
		t.Fatalf("expected Clean 'Hello World', got '%s'", result.Clean)
	}
	if !strings.Contains(result.HTML, `style="color: #74ffff;"`) {
		t.Fatalf("expected HTML to contain #74ffff, got: %s", result.HTML)
	}
}
