package motd

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"mcstatus/internal/types"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var bungeeHexRegex = regexp.MustCompile(`§x(§[0-9a-fA-F]){6}`)

var standardColors = map[string]string{
	"0":             "#000000",
	"black":         "#000000",
	"1":             "#0000aa",
	"dark_blue":     "#0000aa",
	"2":             "#00aa00",
	"dark_green":    "#00aa00",
	"3":             "#00aaaa",
	"dark_aqua":     "#00aaaa",
	"4":             "#aa0000",
	"dark_red":      "#aa0000",
	"5":             "#aa00aa",
	"dark_purple":   "#aa00aa",
	"6":             "#ffaa00",
	"gold":          "#ffaa00",
	"7":             "#aaaaaa",
	"gray":          "#aaaaaa",
	"8":             "#555555",
	"dark_gray":     "#555555",
	"9":             "#5555ff",
	"blue":          "#5555ff",
	"a":             "#55ff55",
	"green":         "#55ff55",
	"b":             "#55ffff",
	"aqua":          "#55ffff",
	"c":             "#ff5555",
	"red":           "#ff5555",
	"d":             "#ff55ff",
	"light_purple":  "#ff55ff",
	"e":             "#ffff55",
	"yellow":        "#ffff55",
	"f":             "#ffffff",
	"white":         "#ffffff",
	"g":             "#ddd605",
	"minecoin_gold": "#ddd605",
}

type TextSpan struct {
	Text          string
	Color         string
	Bold          bool
	Italic        bool
	Underlined    bool
	Strikethrough bool
	Obfuscated    bool
}

// ParseSpans parses any Minecraft MOTD into a slice of TextSpan.
func ParseSpans(desc any) []TextSpan {
	if desc == nil {
		return nil
	}

	var spans []TextSpan

	switch v := desc.(type) {
	case string:
		spans = parseLegacyString(v)
	case map[string]any:
		parseComponent(v, nil, &spans)
	case []any:
		for _, item := range v {
			parseComponent(item, nil, &spans)
		}
	default:
		spans = parseLegacyString(fmt.Sprint(v))
	}
	return spans
}

// Format formats any Minecraft MOTD (Chat Component map, array, or legacy string)
// into a types.FormattedString supporting 1.16+ RGB Hex colors.
func Format(desc any) *types.FormattedString {
	if desc == nil {
		return &types.FormattedString{}
	}

	spans := ParseSpans(desc)
	clean := buildClean(spans)
	html := buildHTML(spans)
	raw := buildRaw(spans)

	return &types.FormattedString{
		Raw:   raw,
		Clean: clean,
		HTML:  html,
	}
}

func parseComponent(comp any, parentProps map[string]any, spans *[]TextSpan) {
	if comp == nil {
		return
	}

	switch v := comp.(type) {
	case string:
		*spans = append(*spans, parseLegacyString(v)...)
	case map[string]any:
		props := make(map[string]any)
		for k, val := range parentProps {
			props[k] = val
		}
		for k, val := range v {
			if k != "extra" {
				props[k] = val
			}
		}

		if text, ok := props["text"].(string); ok && text != "" {
			span := TextSpan{
				Text:          text,
				Color:         resolveColor(props["color"]),
				Bold:          toBool(props["bold"]),
				Italic:        toBool(props["italic"]),
				Underlined:    toBool(props["underlined"]),
				Strikethrough: toBool(props["strikethrough"]),
				Obfuscated:    toBool(props["obfuscated"]),
			}
			*spans = append(*spans, span)
		}

		if extra, ok := v["extra"].([]any); ok {
			for _, child := range extra {
				parseComponent(child, props, spans)
			}
		}
	case []any:
		for _, child := range v {
			parseComponent(child, parentProps, spans)
		}
	}
}

func parseLegacyString(text string) []TextSpan {
	// First convert BungeeCord hex §x§r§r§g§g§b§b to #RRGGBB
	text = bungeeHexRegex.ReplaceAllStringFunc(text, func(m string) string {
		hexStr := strings.ReplaceAll(m, "§x", "#")
		hexStr = strings.ReplaceAll(hexStr, "§", "")
		return "§" + hexStr
	})

	var spans []TextSpan
	var cur TextSpan
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '§' && i+1 < len(runes) {
			next := runes[i+1]
			// Check if hex color follows: §#RRGGBB
			if next == '#' && i+7 < len(runes) {
				potentialHex := string(runes[i+1 : i+8])
				if hexColorRegex.MatchString(potentialHex) {
					if cur.Text != "" {
						spans = append(spans, cur)
						cur.Text = ""
					}
					cur.Color = strings.ToLower(potentialHex)
					i += 7
					continue
				}
			}

			code := strings.ToLower(string(next))
			if hex, ok := standardColors[code]; ok {
				if cur.Text != "" {
					spans = append(spans, cur)
					cur.Text = ""
				}
				cur.Color = hex
				cur.Bold = false
				cur.Italic = false
				cur.Underlined = false
				cur.Strikethrough = false
				cur.Obfuscated = false
				i++
				continue
			}

			switch code {
			case "k":
				cur.Obfuscated = true
			case "l":
				cur.Bold = true
			case "m":
				cur.Strikethrough = true
			case "n":
				cur.Underlined = true
			case "o":
				cur.Italic = true
			case "r":
				if cur.Text != "" {
					spans = append(spans, cur)
					cur.Text = ""
				}
				cur = TextSpan{}
			}
			i++
			continue
		}

		cur.Text += string(r)
	}

	if cur.Text != "" {
		spans = append(spans, cur)
	}

	return spans
}

func resolveColor(c any) string {
	if c == nil {
		return ""
	}
	s := strings.ToLower(strings.TrimSpace(fmt.Sprint(c)))
	if hex, ok := standardColors[s]; ok {
		return hex
	}
	if strings.HasPrefix(s, "#") {
		return s
	}
	return ""
}

func toBool(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return strings.EqualFold(s, "true")
	}
	return false
}

func buildClean(spans []TextSpan) string {
	var sb strings.Builder
	for _, s := range spans {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

func buildRaw(spans []TextSpan) string {
	var sb strings.Builder
	for _, s := range spans {
		if s.Color != "" {
			sb.WriteString("§" + s.Color)
		}
		if s.Bold {
			sb.WriteString("§l")
		}
		if s.Italic {
			sb.WriteString("§o")
		}
		if s.Underlined {
			sb.WriteString("§n")
		}
		if s.Strikethrough {
			sb.WriteString("§m")
		}
		if s.Obfuscated {
			sb.WriteString("§k")
		}
		sb.WriteString(s.Text)
	}
	return sb.String()
}

func buildHTML(spans []TextSpan) string {
	var sb strings.Builder
	sb.WriteString("<span>")
	for _, span := range spans {
		var styles []string
		var classes []string

		if span.Color != "" {
			styles = append(styles, fmt.Sprintf("color: %s;", span.Color))
		}
		if span.Bold {
			styles = append(styles, "font-weight: bold;")
		}
		if span.Italic {
			styles = append(styles, "font-style: italic;")
		}
		if span.Underlined {
			styles = append(styles, "text-decoration: underline;")
		}
		if span.Strikethrough {
			styles = append(styles, "text-decoration: line-through;")
		}
		if span.Obfuscated {
			classes = append(classes, "minecraft-format-obfuscated")
		}

		escapedText := html.EscapeString(span.Text)

		if len(styles) == 0 && len(classes) == 0 {
			sb.WriteString(fmt.Sprintf("<span>%s</span>", escapedText))
		} else {
			attr := ""
			if len(classes) > 0 {
				attr += fmt.Sprintf(" class=\"%s\"", strings.Join(classes, " "))
			}
			if len(styles) > 0 {
				attr += fmt.Sprintf(" style=\"%s\"", strings.Join(styles, " "))
			}
			sb.WriteString(fmt.Sprintf("<span%s>%s</span>", attr, escapedText))
		}
	}
	sb.WriteString("</span>")
	return sb.String()
}
