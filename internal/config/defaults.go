package config

import "time"

const (
	DefaultCacheTTL         = 300           // 5 minutes
	DefaultTimeout          = 15            // seconds
	DefaultSearchLimit      = 5
	DefaultSearXNGURL       = "http://localhost:8888"
	DefaultMinContentLength = 50            // words
	MaxRedirects            = 5
	SubprocessTimeout       = 60 * time.Second
	HealthCheckTimeout      = 3 * time.Second
)

// File extensions supported by Markitdown subprocess
var SupportedFileExts = []string{
	".pdf", ".docx", ".doc", ".pptx", ".ppt", ".xlsx", ".xls",
	".html", ".htm", ".md", ".txt", ".json", ".xml", ".csv",
}

// Text file extensions that don't need conversion, just read directly
var TextFileExts = []string{
	".md", ".txt", ".json", ".xml", ".csv",
}
