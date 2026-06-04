package theme

// Theme defines the visual style for rendering context window diagrams.
// The default theme is inspired by monochrome 80s computer interfaces
// (classic Mac OS, VT100 terminals) with subtle color accents.
type Theme struct {
	Name string `yaml:"name"`

	// Canvas dimensions and spacing
	CanvasPadding  int `yaml:"canvas_padding"`
	RegionGap      int `yaml:"region_gap"`
	SubregionGap   int `yaml:"subregion_gap"`
	WindowGap      int `yaml:"window_gap"`

	// Typography
	FontFamily    string  `yaml:"font_family"`
	FontSize      float64 `yaml:"font_size"`
	FontSizeSmall float64 `yaml:"font_size_small"`
	FontSizeTiny  float64 `yaml:"font_size_tiny"`

	// Window frame
	WindowBorderWidth int    `yaml:"window_border_width"`
	WindowBorderColor string `yaml:"window_border_color"`
	WindowCornerRadius int   `yaml:"window_corner_radius"`
	TitleBarHeight    int    `yaml:"title_bar_height"`
	TitleBarFill      string `yaml:"title_bar_fill"`

	// Region styling
	RegionCornerRadius int    `yaml:"region_corner_radius"`
	RegionBorderWidth  int    `yaml:"region_border_width"`
	RegionBorderColor  string `yaml:"region_border_color"`

	// Color map: maps DSL color names to hex colors
	Colors map[string]ColorSet `yaml:"colors"`

	// Overflow pattern
	OverflowDashArray string `yaml:"overflow_dash_array"`
	OverflowColor     string `yaml:"overflow_color"`

	// Warning styling
	WarningColor  string `yaml:"warning_color"`
	WarningSymbol string `yaml:"warning_symbol"`

	// Note styling
	NoteColor string `yaml:"note_color"`
}

// ColorSet defines the fill and text color for a named color.
type ColorSet struct {
	Fill      string `yaml:"fill"`
	Text      string `yaml:"text"`
	Subtext   string `yaml:"subtext"`
	Border    string `yaml:"border,omitempty"`
}

// DefaultTheme returns the classic Mac/80s monochrome theme with color accents.
func DefaultTheme() *Theme {
	return &Theme{
		Name: "macintosh-84",

		CanvasPadding:  40,
		RegionGap:      2,
		SubregionGap:   1,
		WindowGap:      40,

		FontFamily:    "SF Mono, Menlo, Monaco, Consolas, monospace",
		FontSize:      12,
		FontSizeSmall: 10,
		FontSizeTiny:  8,

		WindowBorderWidth:  2,
		WindowBorderColor:  "#000000",
		WindowCornerRadius: 0,
		TitleBarHeight:     24,
		TitleBarFill:       "#000000",

		RegionCornerRadius: 0,
		RegionBorderWidth:  1,
		RegionBorderColor:  "#555555",

		Colors: map[string]ColorSet{
			"primary": {
				Fill:    "#FFFFFF",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#000000",
			},
			"secondary": {
				Fill:    "#E8E8E8",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#000000",
			},
			"accent": {
				Fill:    "#000000",
				Text:    "#FFFFFF",
				Subtext: "#AAAAAA",
				Border:  "#000000",
			},
			"tool": {
				Fill:    "#D4D4D4",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#333333",
			},
			"tool-light": {
				Fill:    "#EEEEEE",
				Text:    "#333333",
				Subtext: "#777777",
				Border:  "#AAAAAA",
			},
			"knowledge": {
				Fill:    "#C8C8C8",
				Text:    "#000000",
				Subtext: "#444444",
				Border:  "#222222",
			},
			"knowledge-light": {
				Fill:    "#E0E0E0",
				Text:    "#333333",
				Subtext: "#777777",
				Border:  "#999999",
			},
			"highlight": {
				Fill:    "#FFE14D",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#000000",
			},
			"empty": {
				Fill:    "none",
				Text:    "#999999",
				Subtext: "#BBBBBB",
				Border:  "#CCCCCC",
			},
			"cycle": {
				Fill:    "#F0F0F0",
				Text:    "#000000",
				Subtext: "#666666",
				Border:  "#000000",
			},
			"think": {
				Fill:    "#B8E0FF",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#4488CC",
			},
			"act": {
				Fill:    "#FFD4B8",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#CC8844",
			},
			"observe": {
				Fill:    "#D4EED4",
				Text:    "#000000",
				Subtext: "#555555",
				Border:  "#44AA44",
			},
		},

		OverflowDashArray: "4,3",
		OverflowColor:     "#CC0000",

		WarningColor:  "#CC0000",
		WarningSymbol: "⚠",

		NoteColor: "#666666",
	}
}

// GetColor returns the ColorSet for a named color, falling back to "primary".
func (t *Theme) GetColor(name string) ColorSet {
	if cs, ok := t.Colors[name]; ok {
		return cs
	}
	return t.Colors["primary"]
}
