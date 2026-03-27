package reader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReaderResult_RenderMarkdown(t *testing.T) {
	result := &ReaderResult{
		Source:      "https://example.com/article",
		Title:       "测试文章",
		Content:     "正文内容段落一。\n\n正文内容段落二。",
		Format:      "markdown",
		FetchedAt:   time.Date(2026, 3, 27, 10, 30, 0, 0, time.UTC),
		WordCount:   100,
		ContentType: "article",
		ExtractMode: "main",
		Metadata: map[string]string{
			"author":  "作者名",
			"site_name": "站点名",
		},
	}

	output := result.RenderMarkdown()

	// Check comment headers
	assert.Contains(t, output, "<!-- source: https://example.com/article -->")
	assert.Contains(t, output, "<!-- title: 测试文章 -->")
	assert.Contains(t, output, "<!-- fetched: 2026-03-27T10:30:00Z -->")
	assert.Contains(t, output, "<!-- words: 100 -->")
	assert.Contains(t, output, "<!-- type: article -->")
	assert.Contains(t, output, "<!-- extract_mode: main -->")

	// Check title heading
	assert.Contains(t, output, "# 测试文章")

	// Check metadata
	assert.Contains(t, output, "> **author:** 作者名")

	// Check content
	assert.Contains(t, output, "正文内容段落一")
}

func TestReaderResult_RenderJSON(t *testing.T) {
	result := &ReaderResult{
		Source:    "https://example.com/article",
		Title:     "测试文章",
		Content:   "正文内容",
		WordCount: 50,
	}

	output := result.RenderJSON()

	assert.Contains(t, output, `"ok": true`)
	assert.Contains(t, output, `"source": "https://example.com/article"`)
	assert.Contains(t, output, `"title": "测试文章"`)
	assert.Contains(t, output, `"word_count": 50`)
}

func TestGuessContentType(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"github", "https://github.com/go-shiori/go-readability", "documentation"},
		{"stackoverflow", "https://stackoverflow.com/questions/123", "forum"},
		{"reddit", "https://reddit.com/r/golang", "forum"},
		{"youtube", "https://youtube.com/watch?v=abc", "video"},
		{"bilibili", "https://bilibili.com/video/BV1xx", "video"},
		{"twitter", "https://twitter.com/user/status/123", "social"},
		{"x.com", "https://x.com/user/status/123", "social"},
		{"default article", "https://example.com/blog/post", "article"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GuessContentType(tt.url, "", nil))
		})
	}
}
