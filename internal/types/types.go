package types

type FormattedString struct {
	Raw   string `json:"raw"`
	Clean string `json:"clean"`
	HTML  string `json:"html"`
}

type JavaPlayer struct {
	UUID      string `json:"uuid"`
	NameRaw   string `json:"name_raw"`
	NameClean string `json:"name_clean"`
	NameHTML  string `json:"name_html"`
}

type JavaPlayers struct {
	Online int          `json:"online"`
	Max    int          `json:"max"`
	List   []JavaPlayer `json:"list"`
}

type JavaVersion struct {
	NameRaw   string `json:"name_raw"`
	NameClean string `json:"name_clean"`
	NameHTML  string `json:"name_html"`
	Protocol  int    `json:"protocol"`
}

type Mod struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Plugin struct {
	Name    string  `json:"name"`
	Version *string `json:"version"`
}

type SRVRecord struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type JavaStatusResponse struct {
	Online      bool             `json:"online"`
	Host        string           `json:"host"`
	Port        uint16           `json:"port"`
	IPAddress   *string          `json:"ip_address"`
	EULABlocked bool             `json:"eula_blocked"`
	RetrievedAt int64            `json:"retrieved_at"`
	ExpiresAt   int64            `json:"expires_at"`
	SRVRecord   *SRVRecord       `json:"srv_record"`
	Version     *JavaVersion     `json:"version,omitempty"`
	Players     *JavaPlayers     `json:"players,omitempty"`
	MOTD        *FormattedString `json:"motd,omitempty"`
	Icon        *string          `json:"icon,omitempty"`
	Mods        []Mod            `json:"mods,omitempty"`
	Software    *string          `json:"software,omitempty"`
	Plugins     []Plugin         `json:"plugins,omitempty"`
}

type BedrockVersion struct {
	Name     *string `json:"name"`
	Protocol *int64  `json:"protocol"`
}

type BedrockPlayers struct {
	Online *int64 `json:"online"`
	Max    *int64 `json:"max"`
}

type BedrockStatusResponse struct {
	Online      bool             `json:"online"`
	Host        string           `json:"host"`
	Port        uint16           `json:"port"`
	IPAddress   *string          `json:"ip_address"`
	EULABlocked bool             `json:"eula_blocked"`
	RetrievedAt int64            `json:"retrieved_at"`
	ExpiresAt   int64            `json:"expires_at"`
	Version     *BedrockVersion  `json:"version,omitempty"`
	Players     *BedrockPlayers  `json:"players,omitempty"`
	MOTD        *FormattedString `json:"motd,omitempty"`
	Gamemode    *string          `json:"gamemode,omitempty"`
	ServerID    *string          `json:"server_id,omitempty"`
	Edition     *string          `json:"edition,omitempty"`
}
