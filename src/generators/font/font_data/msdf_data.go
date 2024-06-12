package font_data

type rect struct {
	Left   float32 `json:"left"`
	Top    float32 `json:"top"`
	Right  float32 `json:"right"`
	Bottom float32 `json:"bottom"`
}

type glyph struct {
	Unicode     int32   `json:"unicode"`
	Advance     float32 `json:"advance"`
	PlaneBounds rect    `json:"planeBounds"` // The bounding box of the glyph as it should be placed on the baseline
	AtlasBounds rect    `json:"atlasBounds"` // The bounding box of the glyph in the atlas
}

type atlas struct {
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type metrics struct {
	EmSize             float32 `json:"emSize"`
	LineHeight         float32 `json:"lineHeight"`
	Ascender           float32 `json:"ascender"`
	Descender          float32 `json:"descender"`
	UnderlineY         float32 `json:"underlineY"`
	UnderlineThickness float32 `json:"underlineThickness"`
}

type kerning struct {
	Unicode1 int32   `json:"unicode1"`
	Unicode2 int32   `json:"unicode2"`
	Advance  float32 `json:"advance"`
}

type MsdfData struct {
	Glyphs  []glyph   `json:"glyphs"`
	Atlas   atlas     `json:"atlas"`
	Metrics metrics   `json:"metrics"`
	Kerning []kerning `json:"kerning"`
}
