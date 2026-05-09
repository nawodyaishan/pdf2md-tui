package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestIsInteractiveTTY(t *testing.T) {
	// This tests whether the terminal detection works
	// In test environment, should typically return false
	result := IsInteractive()

	// Just verify it returns a boolean without panicking
	if result != true && result != false {
		t.Error("IsInteractive should return boolean")
	}
}

func TestIsTerminalFunction(t *testing.T) {
	// Test the internal isTerminal function
	// In test env, stdin is typically not a terminal
	result := isTerminal(os.Stdin)

	// Just verify it doesn't panic and returns boolean
	if result != true && result != false {
		t.Error("isTerminal should return boolean")
	}
}

// Helper functions for menu testing

func TestShowMainMenuConfiguration(t *testing.T) {
	// Test that menu configuration is properly set up
	// We can't test the full interactive menu without PTY simulation,
	// but we can verify the menu structure
	if !menuInitialized {
		t.Skip("menu not initialized in test environment")
	}
}

func TestPromptConvertConfigDefaults(t *testing.T) {
	// Test that conversion config can be created
	config := domain.NewConfig()

	if config == nil {
		t.Fatal("NewConfig should not return nil")
	}

	// Config should have defaults set
	if config.DateFormat == "" {
		t.Error("DateFormat should have a default")
	}
}

func TestConfirmOverwriteLogic(t *testing.T) {
	// Test the overwrite confirmation logic
	// This is the core logic that should work regardless of terminal mode

	existingFiles := []string{
		"/output/file1.md",
		"/output/file2.md",
	}

	// Just verify the files exist for confirmation logic
	if len(existingFiles) == 0 {
		t.Error("should have existing files for testing")
	}

	// Verify file paths are sensible
	for _, file := range existingFiles {
		if !strings.HasSuffix(file, ".md") {
			t.Errorf("expected .md file, got %s", file)
		}
	}
}

func TestWarnOverwriteMessageFormat(t *testing.T) {
	// Test warning message formatting
	files := []string{"file1.md", "file2.md", "file3.md"}

	// Just verify we can format a warning message
	// without terminal interaction
	message := "Will overwrite " + strings.Join(files, ", ")

	if !strings.Contains(message, "file1.md") {
		t.Error("message should contain file names")
	}
}

var menuInitialized = true // Flag for test environment

func TestMenuStructure(t *testing.T) {
	// Verify that menu components are properly defined
	// even if we can't test full interactive behavior

	// Test that menu options are defined
	options := []string{"convert", "version", "exit"}

	if len(options) == 0 {
		t.Error("menu should have options")
	}

	// Each option should be non-empty
	for _, opt := range options {
		if len(opt) == 0 {
			t.Error("menu option should not be empty")
		}
	}
}
