package reader

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReaderResult is the final output structure.
type ReaderResult struct {
	Source      string            `json:"source"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Format      string            `json:"format"`
	FetchedAt   time.Time         `json:"fetched_at"`
	WordCount   int               `json:"word_count"`
	ContentType string            `json:"content_type"`
	ExtractMode string            `json:"extract_mode"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RenderMarkdown outputs the result in Markdown format with comment headers.
func (r *ReaderResult) RenderMarkdown() string {
	var sb strings.Builder

	// Comment header for agent consumption
	sb.WriteString(fmt.Sprintf("<!-- source: %s -->\n", r.Source))
	sb.WriteString(fmt.Sprintf("<!-- title: %s -->\n", r.Title))
	sb.WriteString(fmt.Sprintf("<!-- fetched: %s -->\n", r.FetchedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("<!-- words: %d -->\n", r.WordCount))
	if r.ContentType != "" {
		sb.WriteString(fmt.Sprintf("<!-- type: %s -->\n", r.ContentType))
	}
	if r.ExtractMode != "" {
		sb.WriteString(fmt.Sprintf("<!-- extract_mode: %s -->\n", r.ExtractMode))
	}
	sb.WriteString("\n")

	// Title as heading
	if r.Title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", r.Title))
	}

	// Metadata block
	if len(r.Metadata) > 0 {
		for k, v := range r.Metadata {
			if v != "" {
				sb.WriteString(fmt.Sprintf("> **%s:** %s\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	// Content
	sb.WriteString(r.Content)
	sb.WriteString("\n")

	return sb.String()
}

// RenderJSON outputs the result as JSON.
func (r *ReaderResult) RenderJSON() string {
	resp := struct {
		OK     bool          `json:"ok"`
		Result *ReaderResult `json:"result"`
	}{
		OK:     true,
		Result: r,
	}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		// Fallback to compact JSON
		data, _ = json.Marshal(resp)
	}
	return string(data)
}

// GuessContentType attempts to classify the page type based on metadata.
func GuessContentType(url, siteName string, metadata map[string]string) string {
	urlLower := strings.ToLower(url)

	// Check common patterns
	switch {
	case strings.Contains(urlLower, "github.com"), strings.Contains(urlLower, "gitlab.com"):
		return "documentation"
	case strings.Contains(urlLower, "stackoverflow.com"), strings.Contains(urlLower, "reddit.com"):
		return "forum"
	case strings.Contains(urlLower, "amazon."), strings.Contains(urlLower, "taobao.com"), strings.Contains(urlLower, "jd.com"):
		return "product"
	case strings.Contains(urlLower, "youtube.com"), strings.Contains(urlLower, "bilibili.com"):
		return "video"
	case strings.Contains(urlLower, "twitter.com"), strings.Contains(urlLower, "x.com"), strings.Contains(urlLower, "weibo.com"):
		return "social"
	}

	return "article"
}
