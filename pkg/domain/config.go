package domain

// Config holds the extraction and processing settings.
type Config struct {
	DateFormat    string
	StripNoise    bool
	ExtractImages bool
	Workers       int
	Recursive     bool
	Force         bool
	Verbose       bool
}

// NewConfig returns a Config with default values.
func NewConfig() *Config {
	return &Config{
		DateFormat: "2006-01-02",
	}
}
