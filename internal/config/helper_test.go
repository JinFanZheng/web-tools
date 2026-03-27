package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandHome(t *testing.T) {
	result := expandHome("~/foo/bar")
	assert.Contains(t, result, "foo/bar")
	assert.NotContains(t, result, "~")
}

func TestExpandHome_NoTilde(t *testing.T) {
	result := expandHome("/abs/path")
	assert.Equal(t, "/abs/path", result)
}

func TestSupportedFileExts(t *testing.T) {
	assert.Contains(t, SupportedFileExts, ".pdf")
	assert.Contains(t, SupportedFileExts, ".docx")
	assert.Contains(t, SupportedFileExts, ".txt")
	assert.Contains(t, SupportedFileExts, ".csv")
}

func TestTextFileExts(t *testing.T) {
	assert.Contains(t, TextFileExts, ".md")
	assert.Contains(t, TextFileExts, ".txt")
	assert.Contains(t, TextFileExts, ".json")
	assert.NotContains(t, TextFileExts, ".pdf")
}
