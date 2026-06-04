package dsl

import (
	"fmt"
	"strings"
)

// Size represents a token count that can be expressed in shorthand.
// Supported formats: "128k", "4k", "2048", "0".
type Size int

// ParseSize parses a size string like "128k", "4k", "2048" into an int token count.
func ParseSize(s string) (Size, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	multiplier := 1
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "k") {
		multiplier = 1024
		lower = strings.TrimSuffix(lower, "k")
	}
	var val int
	_, err := fmt.Sscanf(lower, "%d", &val)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}
	return Size(val * multiplier), nil
}

// MustParseSize parses a size string or panics.
func MustParseSize(s string) Size {
	sz, err := ParseSize(s)
	if err != nil {
		panic(err)
	}
	return sz
}

// String returns a human-friendly representation of the size.
func (s Size) String() string {
	if s%1024 == 0 && s >= 1024 {
		return fmt.Sprintf("%dk", s/1024)
	}
	return fmt.Sprintf("%d", int(s))
}

// MarshalYAML implements yaml.Marshaler for Size.
func (s Size) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for Size.
func (s *Size) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		// Try as int
		var intVal int
		if err2 := unmarshal(&intVal); err2 != nil {
			return fmt.Errorf("size must be a string (e.g. '128k') or integer: %w", err)
		}
		*s = Size(intVal)
		return nil
	}
	parsed, err := ParseSize(raw)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// Region represents a single region within a context window.
type Region struct {
	Name        string   `yaml:"name"`
	Size        Size     `yaml:"size"`
	Color       string   `yaml:"color,omitempty"`
	Icon        string   `yaml:"icon,omitempty"`
	Note        string   `yaml:"note,omitempty"`
	Warning     string   `yaml:"warning,omitempty"`
	Overflow    bool     `yaml:"overflow,omitempty"`
	Subregions  []Region `yaml:"subregions,omitempty"`
}

// Window represents a single context window diagram.
type Window struct {
	Name            string   `yaml:"name,omitempty"`
	Size            Size     `yaml:"size"`
	Title           string   `yaml:"title,omitempty"`
	ShowPercentages  *bool   `yaml:"show_percentages,omitempty"`
	ShowTokenCounts  *bool   `yaml:"show_token_counts,omitempty"`
	Regions         []Region `yaml:"regions"`
}

// Diagram is the top-level document. It supports either a single window
// or multiple windows with layout instructions.
type Diagram struct {
	Windows []Window `yaml:"windows,omitempty"`
	Window  *Window  `yaml:"window,omitempty"`
	Layout  string   `yaml:"layout,omitempty"` // "side-by-side" or "stacked"
	Title   string   `yaml:"title,omitempty"`
}

// Validate checks the diagram for consistency.
func (d *Diagram) Validate() error {
	if d.Window == nil && len(d.Windows) == 0 {
		return fmt.Errorf("diagram must have either 'window' or 'windows'")
	}
	if d.Window != nil && len(d.Windows) > 0 {
		return fmt.Errorf("diagram must have either 'window' or 'windows', not both")
	}
	if d.Window != nil {
		return validateWindow(d.Window)
	}
	for i := range d.Windows {
		if err := validateWindow(&d.Windows[i]); err != nil {
			return fmt.Errorf("window %d: %w", i, err)
		}
	}
	return nil
}

func validateWindow(w *Window) error {
	if w.Size <= 0 {
		return fmt.Errorf("window size must be positive, got %s", w.Size)
	}
	used := Size(0)
	for _, r := range w.Regions {
		if r.Size < 0 {
			return fmt.Errorf("region %q has negative size", r.Name)
		}
		used += r.Size
		// Validate subregions
		subUsed := Size(0)
		for _, sr := range r.Subregions {
			if sr.Size < 0 {
				return fmt.Errorf("subregion %q of %q has negative size", sr.Name, r.Name)
			}
			subUsed += sr.Size
		}
		if len(r.Subregions) > 0 && subUsed > r.Size {
			return fmt.Errorf("subregions of %q total %s exceed region size %s", r.Name, subUsed, r.Size)
		}
	}
	if used > w.Size {
		return fmt.Errorf("regions total %s exceed window size %s", used, w.Size)
	}
	return nil
}

// GetWindows returns all windows in the diagram as a slice.
func (d *Diagram) GetWindows() []Window {
	if d.Window != nil {
		return []Window{*d.Window}
	}
	return d.Windows
}
