package spotify

type Device struct {
	ID            string `json:"id"`
	IsActive      bool   `json:"is_active"`
	IsPrivate     bool   `json:"is_private_session"`
	IsRestricted  bool   `json:"is_restricted"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	VolumePercent int    `json:"volume_percent"`
}

type PlaybackState struct {
	Device      Device `json:"device"`
	IsPlaying   bool   `json:"is_playing"`
	ProgressMs  int    `json:"progress_ms"`
	Shuffle     bool   `json:"shuffle_state"`
	RepeatState string `json:"repeat_state"`
	Item        Track  `json:"item"`
}

type Track struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	URI     string   `json:"uri"`
	Type    string   `json:"type"`
	Album   Album    `json:"album"`
	Artists []Artist `json:"artists"`
}

type Album struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

func (t Track) DisplayArtists() string {
	if len(t.Artists) == 0 {
		return ""
	}
	out := t.Artists[0].Name
	for i := 1; i < len(t.Artists); i++ {
		out += ", " + t.Artists[i].Name
	}
	return out
}

func (t Track) DisplayName() string {
	if t.Name == "" {
		return t.URI
	}
	return t.Name
}
