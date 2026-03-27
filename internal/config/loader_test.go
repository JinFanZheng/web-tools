package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, DefaultCacheTTL, cfg.Reader.CacheTTL)
	assert.Equal(t, DefaultTimeout, cfg.Reader.DefaultTimeout)
	assert.Equal(t, true, cfg.Reader.BrowserFallback)
	assert.Equal(t, "markitdown", cfg.Reader.MarkitdownPath)
	assert.Equal(t, "agent-browser", cfg.Reader.AgentBrowserPath)
	assert.Equal(t, DefaultMinContentLength, cfg.Reader.MinContentLength)
	assert.Equal(t, DefaultSearXNGURL, cfg.Search.SearXNGURL)
	assert.Equal(t, DefaultSearchLimit, cfg.Search.DefaultLimit)
	assert.Equal(t, "auto", cfg.Search.DefaultLocale)
}

func TestLoad_WithNoConfigFiles(t *testing.T) {
	// In a clean temp dir, Load should return defaults
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, DefaultSearXNGURL, cfg.Search.SearXNGURL)
	assert.Equal(t, DefaultSearchLimit, cfg.Search.DefaultLimit)
}

func TestLoad_WithLocalConfigFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	localCfg := Config{
		Search: SearchConfig{
			SearXNGURL:   "http://custom:9999",
			DefaultLimit: 10,
		},
	}
	data, _ := json.Marshal(localCfg)
	os.WriteFile("web-tools.json", data, 0644)

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://custom:9999", cfg.Search.SearXNGURL)
	assert.Equal(t, 10, cfg.Search.DefaultLimit)
}

func TestLoad_LocalOverridesUser(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create user config dir
	userDir := filepath.Join(dir, ".config", "web-tools")
	os.MkdirAll(userDir, 0755)

	userCfg := Config{Search: SearchConfig{SearXNGURL: "http://user:8888", DefaultLimit: 3}}
	data, _ := json.Marshal(userCfg)
	os.WriteFile(filepath.Join(userDir, "config.json"), data, 0644)

	// Override HOME to point to temp dir
	t.Setenv("HOME", dir)

	// Create local config that overrides
	localCfg := Config{Search: SearchConfig{SearXNGURL: "http://local:9999"}}
	data, _ = json.Marshal(localCfg)
	os.WriteFile("web-tools.json", data, 0644)

	cfg, err := Load()
	assert.NoError(t, err)
	// Local overrides user
	assert.Equal(t, "http://local:9999", cfg.Search.SearXNGURL)
	// Local only overrides what it sets, user's limit stays
	assert.Equal(t, 3, cfg.Search.DefaultLimit)
}

func TestLoad_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("SEARXNG_URL", "http://env:7777")
	t.Setenv("WEB_READER_CACHE_TTL", "600")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://env:7777", cfg.Search.SearXNGURL)
	assert.Equal(t, 600, cfg.Reader.CacheTTL)
}

func TestLoad_EnvOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	localCfg := Config{Search: SearchConfig{SearXNGURL: "http://file:8888"}}
	data, _ := json.Marshal(localCfg)
	os.WriteFile("web-tools.json", data, 0644)

	t.Setenv("SEARXNG_URL", "http://env:7777")

	cfg, err := Load()
	assert.NoError(t, err)
	// Env overrides file
	assert.Equal(t, "http://env:7777", cfg.Search.SearXNGURL)
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("web-tools.json", []byte("not json"), 0644)

	cfg, err := Load()
	// Should still work, just skip invalid file
	assert.NoError(t, err)
	assert.Equal(t, DefaultSearXNGURL, cfg.Search.SearXNGURL)
}
