package domain

// ExtractionProfile defines the mode for PDF parsing.
type ExtractionProfile string

const (
	ProfileDefault  ExtractionProfile = "default"
	ProfileAcademic ExtractionProfile = "academic"
	ProfileLegacy   ExtractionProfile = "legacy"
)

// Config holds the extraction and processing settings.
type Config struct {
	DateFormat       string
	StripNoise       bool
	ExtractImages    bool
	EntropyThreshold float64
	Profile          ExtractionProfile
	Workers          int
	Recursive        bool
	Force            bool
	Verbose          bool
}

// NewConfig returns a Config with default values.
func NewConfig() *Config {
	return &Config{
		DateFormat: "2006-01-02",
	}
}
