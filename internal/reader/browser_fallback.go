package reader

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
)

// BrowserFallback handles browser-based content extraction via agent-browser subprocess.
type BrowserFallback struct {
	agentBrowserPath string
	timeout          int // seconds
}

// NewBrowserFallback creates a new BrowserFallback.
func NewBrowserFallback(cfg config.ReaderConfig) *BrowserFallback {
	path := cfg.AgentBrowserPath
	if path == "" {
		path = "agent-browser"
	}
	timeout := cfg.DefaultTimeout * 3 // browser takes longer than HTTP
	if timeout <= 0 {
		timeout = 45
	}
	return &BrowserFallback{
		agentBrowserPath: path,
		timeout:          timeout,
	}
}

// Available checks if agent-browser is available on PATH.
func (b *BrowserFallback) Available() bool {
	_, err := exec.LookPath(b.agentBrowserPath)
	return err == nil
}

// Extract fetches a URL via browser and returns the page text content.
func (b *BrowserFallback) Extract(url string, session string) (string, string, error) {
	if !b.Available() {
		return "", "", apperrors.NewEngineError(
			"agent-browser not found",
			fmt.Sprintf("agent-browser executable not found at %q", b.agentBrowserPath),
			map[string]string{"path": b.agentBrowserPath, "url": url},
			[]string{
				"install agent-browser: npm i -g agent-browser",
				"or: brew install agent-browser",
				"or: cargo install agent-browser",
			},
		)
	}

	// Use context timeout for the entire browser pipeline
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.timeout)*time.Second)
	defer cancel()

	// Step 1: Open URL
	args := []string{"open", url}
	if session != "" {
		args = append([]string{"--session-name", session}, args...)
	}
	if err := b.runWithTimeout(ctx, args...); err != nil {
		b.cleanup(session)
		return "", "", apperrors.NewEngineError(
			"browser failed to open URL",
			err.Error(),
			map[string]string{"url": url, "timeout": fmt.Sprintf("%ds", b.timeout)},
			[]string{"check if the URL is accessible", "check agent-browser version: agent-browser --version"},
		)
	}

	// Step 2: Wait for network idle
	waitArgs := []string{"wait", "--load", "networkidle"}
	if session != "" {
		waitArgs = append([]string{"--session-name", session}, waitArgs...)
	}
	if err := b.runWithTimeout(ctx, waitArgs...); err != nil {
		b.cleanup(session)
		return "", "", apperrors.NewEngineError(
			"browser wait failed (page load timeout)",
			err.Error(),
			map[string]string{"url": url},
			[]string{"the page may be too heavy or blocked", "try again or use a simpler page"},
		)
	}

	// Step 3: Get page title (non-fatal)
	titleArgs := []string{"get", "title"}
	if session != "" {
		titleArgs = append([]string{"--session-name", session}, titleArgs...)
	}
	title := ""
	if titleBytes, err := b.outputWithTimeout(ctx, titleArgs...); err == nil {
		title = strings.TrimSpace(string(titleBytes))
	}

	// Step 4: Get page text content
	textArgs := []string{"get", "text", "body"}
	if session != "" {
		textArgs = append([]string{"--session-name", session}, textArgs...)
	}
	output, err := b.outputWithTimeout(ctx, textArgs...)
	if err != nil {
		b.cleanup(session)
		return "", "", apperrors.NewExtractError(
			"browser text extraction failed",
			err.Error(),
			map[string]string{"url": url},
			[]string{"agent-browser get text body"},
			[]string{"the page content may be empty or blocked"},
		)
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		b.cleanup(session)
		return "", "", apperrors.NewExtractError(
			"browser returned empty content",
			"page loaded but body text is empty",
			map[string]string{"url": url, "title": title},
			[]string{"agent-browser get text body"},
			[]string{"the page may require interaction", "the page may use canvas/iframe"},
		)
	}

	return title, content, nil
}

// runWithTimeout runs an agent-browser command with context timeout.
func (b *BrowserFallback) runWithTimeout(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, b.agentBrowserPath, args...)
	return cmd.Run()
}

// outputWithTimeout runs an agent-browser command and captures stdout with context timeout.
func (b *BrowserFallback) outputWithTimeout(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, b.agentBrowserPath, args...)
	return cmd.Output()
}

// cleanup closes the browser session (best effort).
func (b *BrowserFallback) cleanup(session string) {
	args := []string{"close"}
	if session != "" {
		args = append([]string{"--session-name", session}, args...)
	}
	cmd := exec.Command(b.agentBrowserPath, args...)
	_ = cmd.Run()
}

// Close closes the browser session (public API).
func (b *BrowserFallback) Close(session string) {
	b.cleanup(session)
}
