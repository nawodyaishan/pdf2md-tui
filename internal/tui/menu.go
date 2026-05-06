package tui

import (
	"os"

	"github.com/pterm/pterm"
)

// IsInteractive reports whether stdin is an interactive terminal.
func IsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// MenuConfig holds conversion parameters collected through the interactive menu.
type MenuConfig struct {
	Directory  string
	Recursive  bool
	StripNoise bool
}

const (
	menuConvert = "Convert PDFs"
	menuVersion = "View version"
	menuExit    = "Exit"
)

// ShowMainMenu presents the interactive top-level menu.
// Returns action ("convert", "version", "exit") and any collected config.
func ShowMainMenu() (action string, cfg MenuConfig, err error) {
	choice, err := pterm.DefaultInteractiveSelect.
		WithOptions([]string{menuConvert, menuVersion, menuExit}).
		Show("What would you like to do?")
	if err != nil {
		return "", cfg, err
	}

	switch choice {
	case menuConvert:
		cfg, err = promptConvertConfig()
		return "convert", cfg, err
	case menuVersion:
		return "version", cfg, nil
	default:
		return "exit", cfg, nil
	}
}

func promptConvertConfig() (MenuConfig, error) {
	var cfg MenuConfig

	dir, err := pterm.DefaultInteractiveTextInput.
		WithDefaultValue(".").
		Show("Directory to scan")
	if err != nil {
		return cfg, err
	}
	if dir == "" {
		dir = "."
	}
	cfg.Directory = dir

	const (
		optRecursive  = "Recursive — scan subdirectories"
		optStripNoise = "Strip noise — remove headers/footers for LLM optimization"
	)

	selected, err := pterm.DefaultInteractiveMultiselect.
		WithOptions([]string{optRecursive, optStripNoise}).
		Show("Options")
	if err != nil {
		return cfg, err
	}

	for _, s := range selected {
		switch s {
		case optRecursive:
			cfg.Recursive = true
		case optStripNoise:
			cfg.StripNoise = true
		}
	}

	return cfg, nil
}

// ConfirmOverwrite warns about existing output files and prompts the user to continue.
// Returns true if conversion should proceed.
func (p *Progress) ConfirmOverwrite(files []string) (bool, error) {
	pterm.Println()
	pterm.Warning.Printfln("%d output file(s) already exist and will be overwritten:", len(files))
	for _, f := range files {
		pterm.Println("  •", pterm.Gray(f))
	}
	pterm.Println()
	return pterm.DefaultInteractiveConfirm.
		WithDefaultValue(false).
		Show("Continue and overwrite?")
}

// WarnOverwrite prints a non-interactive overwrite notice for non-TTY environments.
func (p *Progress) WarnOverwrite(files []string) {
	pterm.Warning.Printfln("%d output file(s) already exist and will be overwritten:", len(files))
	for _, f := range files {
		pterm.Println("  •", pterm.Gray(f))
	}
}
