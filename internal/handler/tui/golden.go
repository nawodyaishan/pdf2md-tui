package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// golden provides utility functions for golden file testing.
// Golden files allow verifying that string output (like rendered UI) remains consistent.

// updateGoldenFiles returns true if UPDATE_GOLDEN=1 environment variable is set.
// This allows tests to regenerate golden files when intentional changes are made.
func updateGoldenFiles() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

// CompareGolden compares actual output against a golden file.
// If UPDATE_GOLDEN=1, writes actual output to the golden file.
// Otherwise, compares and fails if output differs.
func CompareGolden(t *testing.T, actual string, goldenFile string) {
	goldenPath := filepath.Join("testdata", "golden", goldenFile)

	// Ensure directory exists
	goldenDir := filepath.Dir(goldenPath)
	if _, err := os.Stat(goldenDir); os.IsNotExist(err) {
		if err := os.MkdirAll(goldenDir, 0755); err != nil {
			t.Fatalf("failed to create golden dir: %v", err)
		}
	}

	if updateGoldenFiles() {
		// Write actual output to golden file
		if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	// Read golden file
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			// First run - create golden file
			if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
				t.Fatalf("failed to write initial golden file: %v", err)
			}
			t.Logf("created initial golden file: %s (run tests again to validate)", goldenPath)
			return
		}
		t.Fatalf("failed to read golden file: %v", err)
	}

	// Compare
	if actual != string(expected) {
		t.Errorf("output mismatch in %s\n\nExpected:\n%s\n\nGot:\n%s\n\nTo update, run: UPDATE_GOLDEN=1 go test ./...",
			goldenFile, string(expected), actual)
	}
}

// StripANSI removes ANSI escape codes from a string for easier golden file comparison.
func StripANSI(s string) string {
	// Simple ANSI code removal for testing
	// Matches common patterns: \033[...m, \x1b[...m
	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		if (s[i] == '\033' || s[i] == '\x1b') && i+1 < len(s) && s[i+1] == '[' {
			// Skip until 'm'
			for i < len(s) && s[i] != 'm' {
				i++
			}
		} else {
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}

// NormalizeOutput prepares output for golden file comparison by:
// - Removing ANSI color codes
// - Normalizing line endings
// - Trimming trailing whitespace
func NormalizeOutput(s string) string {
	s = StripANSI(s)
	// Normalize line endings to \n
	normalized := bytes.ReplaceAll([]byte(s), []byte("\r\n"), []byte("\n"))
	// Remove trailing whitespace from lines
	lines := bytes.Split(normalized, []byte("\n"))
	for i := range lines {
		lines[i] = bytes.TrimRight(lines[i], " \t")
	}
	return string(bytes.Join(lines, []byte("\n")))
}

// WriteGoldenFile writes content to a golden file (for manual updates).
func WriteGoldenFile(t *testing.T, goldenFile string, content string) {
	goldenPath := filepath.Join("testdata", "golden", goldenFile)
	goldenDir := filepath.Dir(goldenPath)

	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}

	if err := os.WriteFile(goldenPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}
}

// ReadGoldenFile reads a golden file for comparison.
func ReadGoldenFile(t *testing.T, goldenFile string) string {
	goldenPath := filepath.Join("testdata", "golden", goldenFile)

	content, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file not found: %s (run UPDATE_GOLDEN=1 go test to create)", goldenFile)
		}
		t.Fatalf("failed to read golden file: %v", err)
	}

	return string(content)
}
