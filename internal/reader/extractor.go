package reader

import (
	"io"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-shiori/go-readability"
	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
)

// ExtractResult holds the extracted content from a web page.
type ExtractResult struct {
	Title         string            `json:"title"`
	Content       string            `json:"content"`        // Markdown format
	TextContent   string            `json:"text_content"`   // Plain text
	HTML          string            `json:"html"`           // Raw readability HTML (for debugging)
	Byline        string            `json:"byline"`
	Excerpt       string            `json:"excerpt"`
	Length        int               `json:"length"`
	SiteName      string            `json:"site_name"`
	PublishedTime *time.Time        `json:"published_time,omitempty"`
	ModifiedTime  *time.Time        `json:"modified_time,omitempty"`
	Image         string            `json:"image,omitempty"`
	Favicon       string            `json:"favicon,omitempty"`
	Language      string            `json:"language,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Extractor extracts readable content from HTML pages.
type Extractor struct {
	minContentLength int
	mdConverter      *md.Converter
}

// NewExtractor creates a new Extractor.
func NewExtractor(cfg config.ReaderConfig) *Extractor {
	converter := md.NewConverter("", true, nil)
	return &Extractor{
		minContentLength: cfg.MinContentLength,
		mdConverter:      converter,
	}
}

// Extract parses HTML and extracts the main readable content.
func (e *Extractor) Extract(body io.Reader, pageURL *url.URL) (*ExtractResult, error) {
	article, err := readability.FromReader(body, pageURL)
	if err != nil {
		return nil, apperrors.NewExtractError(
			"readability 解析失败",
			err.Error(),
			map[string]string{"url": pageURL.String()},
			[]string{"readability"},
			[]string{"尝试 --browser 强制使用浏览器渲染"},
		)
	}

	// Convert readability HTML to Markdown
	mdContent, err := e.mdConverter.ConvertString(article.Content)
	if err != nil {
		// Fallback to TextContent if HTML→MD fails
		mdContent = article.TextContent
	}

	// Clean up excessive whitespace
	mdContent = cleanMarkdown(mdContent)

	result := &ExtractResult{
		Title:         strings.TrimSpace(article.Title),
		Content:       mdContent,
		TextContent:   strings.TrimSpace(article.TextContent),
		HTML:          article.Content,
		Byline:        article.Byline,
		Excerpt:       article.Excerpt,
		Length:        article.Length,
		SiteName:      article.SiteName,
		PublishedTime: article.PublishedTime,
		ModifiedTime:  article.ModifiedTime,
		Image:         article.Image,
		Favicon:       article.Favicon,
		Language:      article.Language,
		Metadata:      make(map[string]string),
	}

	if article.Byline != "" {
		result.Metadata["author"] = article.Byline
	}
	if article.SiteName != "" {
		result.Metadata["site_name"] = article.SiteName
	}
	if article.Excerpt != "" {
		result.Metadata["description"] = article.Excerpt
	}
	if article.Language != "" {
		result.Metadata["language"] = article.Language
	}
	if article.Image != "" {
		result.Metadata["image"] = article.Image
	}
	if article.Favicon != "" {
		result.Metadata["favicon"] = article.Favicon
	}

	return result, nil
}

// NeedsFallback checks if the extraction result is too poor to use.
func (e *Extractor) NeedsFallback(result *ExtractResult) bool {
	words := strings.Fields(result.TextContent)
	if len(words) < e.minContentLength {
		return true
	}
	return false
}

func cleanMarkdown(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}
