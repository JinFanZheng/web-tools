package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Load reads config from files and environment variables, merges with defaults.
// Priority (high to low): env vars > current dir config > user config > defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// 1. Load user config: ~/.config/web-tools/config.json
	userCfg, err := loadConfigFile(expandHome("~/.config/web-tools/config.json"))
	if err == nil {
		mergeReaderConfig(&cfg.Reader, userCfg.Reader)
		mergeSearchConfig(&cfg.Search, userCfg.Search)
	}

	// 2. Load local config: ./web-tools.json
	localCfg, err := loadConfigFile("web-tools.json")
	if err == nil {
		mergeReaderConfig(&cfg.Reader, localCfg.Reader)
		mergeSearchConfig(&cfg.Search, localCfg.Search)
	}

	// 3. Override with environment variables
	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Invalid JSON: skip silently, don't break the flow
		return nil, fmt.Errorf("invalid config JSON in %s: %w", path, err)
	}

	return &cfg, nil
}

func mergeReaderConfig(dst *ReaderConfig, src ReaderConfig) {
	if src.CacheDir != "" {
		dst.CacheDir = src.CacheDir
	}
	if src.CacheTTL != 0 {
		dst.CacheTTL = src.CacheTTL
	}
	if src.DefaultTimeout != 0 {
		dst.DefaultTimeout = src.DefaultTimeout
	}
	if src.MarkitdownPath != "" {
		dst.MarkitdownPath = src.MarkitdownPath
	}
	if src.AgentBrowserPath != "" {
		dst.AgentBrowserPath = src.AgentBrowserPath
	}
	if src.MinContentLength != 0 {
		dst.MinContentLength = src.MinContentLength
	}
	if src.BrowserFallback {
		dst.BrowserFallback = src.BrowserFallback
	}
}

func mergeSearchConfig(dst *SearchConfig, src SearchConfig) {
	if src.SearXNGURL != "" {
		dst.SearXNGURL = src.SearXNGURL
	}
	if src.DefaultLimit != 0 {
		dst.DefaultLimit = src.DefaultLimit
	}
	if src.DefaultLocale != "" {
		dst.DefaultLocale = src.DefaultLocale
	}
	if src.TavilyAPIKey != "" {
		dst.TavilyAPIKey = src.TavilyAPIKey
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SEARXNG_URL"); v != "" {
		cfg.Search.SearXNGURL = v
	}
	if v := os.Getenv("TAVILY_API_KEY"); v != "" {
		cfg.Search.TavilyAPIKey = v
	}
	if v := os.Getenv("WEB_READER_CACHE_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Reader.CacheTTL = n
		}
	}
	if v := os.Getenv("WEB_READER_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Reader.DefaultTimeout = n
		}
	}
	if os.Getenv("WEB_READER_NO_BROWSER") != "" {
		cfg.Reader.BrowserFallback = false
	}
	if v := os.Getenv("MARKITDOWN_PATH"); v != "" {
		cfg.Reader.MarkitdownPath = v
	}
	if v := os.Getenv("WEB_TOOLS_CONFIG"); v != "" {
		fileCfg, err := loadConfigFile(v)
		if err == nil {
			mergeReaderConfig(&cfg.Reader, fileCfg.Reader)
			mergeSearchConfig(&cfg.Search, fileCfg.Search)
		}
	}
}
