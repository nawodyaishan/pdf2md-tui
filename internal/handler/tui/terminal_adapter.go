package tui

import (
	"bufio"
	"io"
)

// TerminalReader abstracts reading user input for testability.
type TerminalReader interface {
	// ReadLine reads a line of input
	ReadLine() (string, error)
}

// RealTerminalReader wraps bufio.Scanner for reading from stdin.
type RealTerminalReader struct {
	scanner *bufio.Scanner
}

// NewRealTerminalReader creates a reader wrapping stdin.
func NewRealTerminalReader(input io.Reader) *RealTerminalReader {
	return &RealTerminalReader{
		scanner: bufio.NewScanner(input),
	}
}

// ReadLine reads the next line of input.
func (r *RealTerminalReader) ReadLine() (string, error) {
	if !r.scanner.Scan() {
		return "", r.scanner.Err()
	}
	return r.scanner.Text(), nil
}

// MockTerminalReader provides canned input for testing.
type MockTerminalReader struct {
	lines []string
	idx   int
}

// NewMockTerminalReader creates a reader with predefined input lines.
func NewMockTerminalReader(lines []string) *MockTerminalReader {
	return &MockTerminalReader{
		lines: lines,
		idx:   0,
	}
}

// ReadLine returns the next predefined line.
func (r *MockTerminalReader) ReadLine() (string, error) {
	if r.idx >= len(r.lines) {
		return "", io.EOF
	}
	line := r.lines[r.idx]
	r.idx++
	return line, nil
}

// TerminalWriter abstracts writing output for testability.
type TerminalWriter interface {
	// WriteLine writes a line of output
	WriteLine(s string) error
}

// RealTerminalWriter wraps io.Writer for writing to stdout.
type RealTerminalWriter struct {
	writer io.Writer
}

// NewRealTerminalWriter creates a writer wrapping stdout.
func NewRealTerminalWriter(output io.Writer) *RealTerminalWriter {
	return &RealTerminalWriter{
		writer: output,
	}
}

// WriteLine writes a line to the output.
func (w *RealTerminalWriter) WriteLine(s string) error {
	_, err := w.writer.Write([]byte(s + "\n"))
	return err
}

// MockTerminalWriter collects output for verification.
type MockTerminalWriter struct {
	lines []string
}

// NewMockTerminalWriter creates a writer that collects output.
func NewMockTerminalWriter() *MockTerminalWriter {
	return &MockTerminalWriter{
		lines: []string{},
	}
}

// WriteLine adds a line to the collected output.
func (w *MockTerminalWriter) WriteLine(s string) error {
	w.lines = append(w.lines, s)
	return nil
}

// Output returns all collected output as a single string.
func (w *MockTerminalWriter) Output() string {
	result := ""
	for _, line := range w.lines {
		result += line + "\n"
	}
	return result
}
