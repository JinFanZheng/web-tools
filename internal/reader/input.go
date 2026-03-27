package reader

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vanzheng/web-tools/internal/config"
)

// InputType distinguishes between URL and local file inputs.
type InputType int

const (
	InputURL  InputType = iota // Remote URL (http/https)
	InputFile                  // Local file path
)

// Input represents a parsed input source.
type Input struct {
	Type     InputType
	Original string // Original user input
	URL      *url.URL
	FilePath string
}

// ParseInput determines whether the input is a URL or a local file.
func ParseInput(raw string) (*Input, error) {
	raw = strings.TrimSpace(raw)

	// Try URL first
	if parsed, err := url.ParseRequestURI(raw); err == nil {
		scheme := strings.ToLower(parsed.Scheme)
		if scheme == "http" || scheme == "https" {
			return &Input{
				Type:     InputURL,
				Original: raw,
				URL:      parsed,
			}, nil
		}
	}

	// Check if it looks like a file path (absolute or relative)
	if raw != "" {
		// Expand ~ to home directory
		if strings.HasPrefix(raw, "~") {
			home, _ := os.UserHomeDir()
			raw = filepath.Join(home, raw[1:])
		}

		if _, err := os.Stat(raw); err == nil {
			absPath, _ := filepath.Abs(raw)
			return &Input{
				Type:     InputFile,
				Original: raw,
				FilePath: absPath,
			}, nil
		}
	}

	return nil, nil
}

// Extension returns the file extension (with dot, lowercase) for file inputs.
// Returns empty string for URL inputs.
func (i *Input) Extension() string {
	if i.Type != InputFile {
		return ""
	}
	return strings.ToLower(filepath.Ext(i.FilePath))
}

// NeedsConversion checks if a file input needs markitdown conversion.
func (i *Input) NeedsConversion() bool {
	if i.Type != InputFile {
		return false
	}
	ext := i.Extension()
	for _, supported := range config.SupportedFileExts {
		if ext == supported {
			// Text files don't need conversion
			for _, text := range config.TextFileExts {
				if ext == text {
					return false
				}
			}
			return true
		}
	}
	return false
}

// DisplayName returns a human-readable name for the input source.
func (i *Input) DisplayName() string {
	if i.Type == InputURL {
		return i.URL.String()
	}
	return i.FilePath
}
