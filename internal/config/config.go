package config

// Config is the top-level configuration.
type Config struct {
	Reader ReaderConfig `json:"reader"`
	Search SearchConfig `json:"search"`
}

// ReaderConfig holds web-reader specific settings.
type ReaderConfig struct {
	CacheDir         string `json:"cache_dir"`
	CacheTTL         int    `json:"cache_ttl"`
	DefaultTimeout   int    `json:"default_timeout"`
	BrowserFallback  bool   `json:"browser_fallback"`
	MarkitdownPath   string `json:"markitdown_path"`
	AgentBrowserPath string `json:"agent_browser_path"`
	MinContentLength int    `json:"min_content_length"`
}

// SearchConfig holds web-search specific settings.
type SearchConfig struct {
	SearXNGURL    string `json:"searxng_url"`
	TavilyAPIKey  string `json:"tavily_api_key"`
	DefaultLimit  int    `json:"default_limit"`
	DefaultLocale string `json:"default_locale"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Reader: ReaderConfig{
			CacheDir:         expandHome("~/.cache/web-tools"),
			CacheTTL:         DefaultCacheTTL,
			DefaultTimeout:   DefaultTimeout,
			BrowserFallback:  true,
			MarkitdownPath:   "markitdown",
			AgentBrowserPath: "agent-browser",
			MinContentLength: DefaultMinContentLength,
		},
		Search: SearchConfig{
			SearXNGURL:    DefaultSearXNGURL,
			DefaultLimit:  DefaultSearchLimit,
			DefaultLocale: "auto",
		},
	}
}
