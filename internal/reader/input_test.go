package reader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInput_URL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantURL string
		wantErr bool
	}{
		{"https url", "https://example.com/article", "https://example.com/article", false},
		{"http url", "http://localhost:8080/doc", "http://localhost:8080/doc", false},
		{"url with path", "https://github.com/go-shiori/go-readability", "https://github.com/go-shiori/go-readability", false},
		{"url with query", "https://example.com/search?q=test&lang=en", "https://example.com/search?q=test&lang=en", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inp, err := ParseInput(tt.input)
			assert.NoError(t, err)
			assert.NotNil(t, inp)
			assert.Equal(t, InputURL, inp.Type)
			assert.Equal(t, tt.wantURL, inp.URL.String())
		})
	}
}

func TestParseInput_File(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.md")
	assert.NoError(t, os.WriteFile(tmpFile, []byte("# test\n"), 0644))

	inp, err := ParseInput(tmpFile)
	assert.NoError(t, err)
	assert.NotNil(t, inp)
	assert.Equal(t, InputFile, inp.Type)
	assert.Equal(t, ".md", inp.Extension())
	assert.False(t, inp.NeedsConversion())
	assert.Equal(t, tmpFile, inp.FilePath)
}

func TestParseInput_Invalid(t *testing.T) {
	inp, err := ParseInput("not-a-url-and-not-a-file-xyz")
	assert.NoError(t, err)
	assert.Nil(t, inp)
}

func TestInput_NeedsConversion(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"doc.pdf", true},
		{"doc.docx", true},
		{"doc.xlsx", true},
		{"doc.md", false},
		{"doc.txt", false},
		{"doc.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)
			assert.NoError(t, os.WriteFile(tmpFile, []byte("x"), 0644))

			inp, _ := ParseInput(tmpFile)
			assert.Equal(t, tt.want, inp.NeedsConversion())
		})
	}
}

func TestInput_DisplayName(t *testing.T) {
	urlInput, _ := ParseInput("https://example.com")
	assert.Equal(t, "https://example.com", urlInput.DisplayName())

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.md")
	os.WriteFile(tmpFile, []byte("x"), 0644)
	fileInput, _ := ParseInput(tmpFile)
	assert.Contains(t, fileInput.DisplayName(), "test.md")
}

func TestInput_URLExtension(t *testing.T) {
	urlInput, _ := ParseInput("https://example.com")
	assert.Equal(t, "", urlInput.Extension())
}
