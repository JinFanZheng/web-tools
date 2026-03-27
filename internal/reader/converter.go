package reader

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
)

// Converter handles file format conversion via markitdown subprocess.
type Converter struct {
	markitdownPath string
	timeout        int // seconds
}

// NewConverter creates a new Converter.
func NewConverter(cfg config.ReaderConfig) *Converter {
	path := cfg.MarkitdownPath
	if path == "" {
		path = "markitdown"
	}
	return &Converter{
		markitdownPath: path,
		timeout:        int(config.SubprocessTimeout.Seconds()),
	}
}

// Available checks if markitdown is available on PATH.
func (c *Converter) Available() bool {
	_, err := exec.LookPath(c.markitdownPath)
	return err == nil
}

// Convert converts a file to Markdown using markitdown.
func (c *Converter) Convert(filePath string) (string, error) {
	if !c.Available() {
		return "", apperrors.NewEngineError(
			"markitdown not found",
			fmt.Sprintf("markitdown executable not found at %q", c.markitdownPath),
			map[string]string{"path": c.markitdownPath, "file": filePath},
			[]string{
				"install markitdown: pip install markitdown",
				"or: brew install markitdown (if available)",
				"or set MarkitdownPath in config to the correct path",
			},
		)
	}

	cmd := exec.Command(c.markitdownPath, filePath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", apperrors.NewEngineError(
				"markitdown conversion failed",
				fmt.Sprintf("exit code %d, stderr: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
				map[string]string{"file": filePath, "path": c.markitdownPath},
				[]string{"check if the file is a supported format", "check markitdown version: markitdown --version"},
			)
		}
		return "", apperrors.NewEngineError(
			"markitdown execution failed",
			err.Error(),
			map[string]string{"file": filePath, "path": c.markitdownPath},
			nil,
		)
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return "", apperrors.NewExtractError(
			"markitdown returned empty content",
			"conversion succeeded but output is empty",
			map[string]string{"file": filePath},
			[]string{"markitdown"},
			[]string{"the file might be empty or unsupported"},
		)
	}

	return result, nil
}
