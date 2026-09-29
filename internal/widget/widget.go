package widget

import (
	"image"
	"strings"
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
}

// Render generates a PNG image based on data.Style.
func Render(data *WidgetData) ([]byte, error) {
	if data == nil {
		data = &WidgetData{
			Style:       "default",
			Host:        "localhost",
			Port:        25565,
			Edition:     "Java Edition",
			Dark:        true,
			Rounded:     true,
			Transparent: false,
		}
	}

	switch strings.ToLower(strings.TrimSpace(data.Style)) {
	case "minecraft", "classic", "mc", "game":
		return RenderMinecraft(data)
	default:
		return RenderDefault(data)
	}
}

// RenderWidget is a convenience wrapper for Render.
func RenderWidget(data *WidgetData) ([]byte, error) {
	return Render(data)
}

// Render on WidgetData allows calling data.Render().
func (d *WidgetData) Render() ([]byte, error) {
	return Render(d)
}
