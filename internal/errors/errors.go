package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// AppError is a structured error for agent consumption.
type AppError struct {
	Code        int               `json:"code"`
	Category    string            `json:"category"`
	Message     string            `json:"message"`
	Detail      string            `json:"detail"`
	Context     map[string]string `json:"context,omitempty"`
	Attempted   []string          `json:"attempted,omitempty"`
	Suggestions []string          `json:"suggestions"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Category, e.Message, e.Detail)
}

// ExitCode returns the process exit code for this error.
func (e *AppError) ExitCode() int {
	return e.Code
}

// ToJSON returns the error as JSON bytes.
func (e *AppError) ToJSON() []byte {
	resp := struct {
		OK    bool      `json:"ok"`
		Error *AppError `json:"error"`
	}{
		OK:    false,
		Error: e,
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return data
}

// ToStderr returns the error as a human-readable Markdown string for stderr.
func (e *AppError) ToStderr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[ERROR] %s: %s\n", e.Category, e.Message))

	for k, v := range e.Context {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	if e.Detail != "" {
		sb.WriteString(fmt.Sprintf("  原因: %s\n", e.Detail))
	}
	if len(e.Attempted) > 0 {
		sb.WriteString(fmt.Sprintf("  已尝试: %s\n", strings.Join(e.Attempted, " → ")))
	}
	for _, s := range e.Suggestions {
		sb.WriteString(fmt.Sprintf("  建议: %s\n", s))
	}
	return sb.String()
}

// HandleError prints the error to stderr and exits with the appropriate code.
func HandleError(err error) {
	if err == nil {
		return
	}
	var appErr *AppError
	if As(err, &appErr) {
		fmt.Fprintln(os.Stderr, appErr.ToStderr())
		os.Exit(appErr.ExitCode())
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// As wraps errors.As for convenience.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Helper constructors

func NewNetworkError(message, detail string, context map[string]string, suggestions []string) *AppError {
	return &AppError{
		Code:        1,
		Category:    "network",
		Message:     message,
		Detail:      detail,
		Context:     context,
		Suggestions: suggestions,
	}
}

func NewUnreachableError(message, detail string, context map[string]string, suggestions []string) *AppError {
	return &AppError{
		Code:        2,
		Category:    "unreachable",
		Message:     message,
		Detail:      detail,
		Context:     context,
		Suggestions: suggestions,
	}
}

func NewExtractError(message, detail string, context map[string]string, attempted []string, suggestions []string) *AppError {
	return &AppError{
		Code:        3,
		Category:    "extract",
		Message:     message,
		Detail:      detail,
		Context:     context,
		Attempted:   attempted,
		Suggestions: suggestions,
	}
}

func NewEngineError(message, detail string, context map[string]string, suggestions []string) *AppError {
	return &AppError{
		Code:        4,
		Category:    "engine",
		Message:     message,
		Detail:      detail,
		Context:     context,
		Suggestions: suggestions,
	}
}

func NewInputError(message, detail string, suggestions []string) *AppError {
	return &AppError{
		Code:        1,
		Category:    "input",
		Message:     message,
		Detail:      detail,
		Suggestions: suggestions,
	}
}
