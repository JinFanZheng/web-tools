package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtractSource_StringArray(t *testing.T) {
	// SearXNG default format: parsed_url is a string array ["https", "host", "/path", ...]
	raw := json.RawMessage(`["https", "example.com", "/article", "", "", ""]`)
	r := SearXNGResult{URL: "https://example.com/article", ParsedURL: raw}
	got := ExtractSource(r)
	assert.Equal(t, "example.com", got)
}

func TestExtractSource_ObjectArray(t *testing.T) {
	// Alternative format: parsed_url is array of objects
	raw := json.RawMessage(`[{"scheme":"https","netloc":"example.com"}]`)
	r := SearXNGResult{URL: "https://example.com/article", ParsedURL: raw}
	got := ExtractSource(r)
	assert.Equal(t, "example.com", got)
}

func TestExtractSource_FallbackToURL(t *testing.T) {
	// No parsed_url, fallback to URL parsing
	r := SearXNGResult{URL: "https://github.com/go-shiori/go-readability"}
	got := ExtractSource(r)
	assert.Equal(t, "github.com", got)
}

func TestExtractSource_Empty(t *testing.T) {
	r := SearXNGResult{URL: "not-a-url"}
	got := ExtractSource(r)
	assert.Equal(t, "unknown", got)
}

func TestSearchResponse_RenderMarkdown(t *testing.T) {
	resp := &SearchResponse{
		Query:  "golang readability",
		Engine: "searxng",
		Locale: "en-US",
		Total:  2,
		Results: []SearchResult{
			{Rank: 1, Title: "go-readability", URL: "https://github.com/go-shiori/go-readability", Snippet: "Extract content from HTML", Source: "github.com", Engines: []string{"google"}},
			{Rank: 2, Title: "Readability.js", URL: "https://github.com/mozilla/readability", Snippet: "Mozilla readability", Source: "github.com", Engines: []string{"bing"}},
		},
		SearchedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	}

	md := resp.RenderMarkdown()
	assert.Contains(t, md, `## Search: "golang readability"`)
	assert.Contains(t, md, "Engine: searxng | Locale: en-US | Results: 2")
	assert.Contains(t, md, "### 1. go-readability")
	assert.Contains(t, md, "### 2. Readability.js")
	assert.Contains(t, md, "**URL:** https://github.com/go-shiori/go-readability")
	assert.Contains(t, md, "**Snippet:** Extract content from HTML")
}

func TestSearchResponse_RenderJSON(t *testing.T) {
	resp := &SearchResponse{
		Query:  "test query",
		Engine: "searxng",
		Total:  1,
		Results: []SearchResult{
			{Rank: 1, Title: "Result", URL: "https://example.com", Snippet: "A snippet"},
		},
		SearchedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	}

	j := resp.RenderJSON()
	assert.Contains(t, j, `"ok": true`)
	assert.Contains(t, j, `"query": "test query"`)
	assert.Contains(t, j, `"rank": 1`)
	assert.Contains(t, j, `"title": "Result"`)
}

func TestSearXNGClient_Query(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/search")
		assert.Equal(t, "json", r.URL.Query().Get("format"))
		assert.Equal(t, "test query", r.URL.Query().Get("q"))
		assert.Equal(t, "general", r.URL.Query().Get("categories"))

		w.Header().Set("Content-Type", "application/json")
		// Simulate real SearXNG string-array parsed_url format
		w.Write([]byte(`{
			"number_of_results": 2,
			"results": [
				{"title": "Result 1", "url": "https://example.com/1", "content": "Snippet 1", "engines": ["google"], "parsed_url": ["https", "example.com", "/1"]},
				{"title": "Result 2", "url": "https://example.com/2", "content": "Snippet 2", "engines": ["bing"], "parsed_url": ["https", "example.com", "/2"]}
			]
		}`))
	}))
	defer server.Close()

	client := NewSearXNGClient(server.URL)
	client.httpClient.Timeout = 5 * time.Second

	opts := SearXNGOptions{Limit: 5, Category: "general"}
	results, err := client.Query("test query", opts)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Result 1", results[0].Title)
	assert.Equal(t, "Snippet 2", results[1].Content)
	assert.Equal(t, "example.com", ExtractSource(results[0]))
}

func TestSearXNGClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSearXNGClient(server.URL)
	assert.NoError(t, client.HealthCheck())
}

func TestSearXNGClient_Query_Limit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"number_of_results": 5,
			"results": [
				{"title": "R1", "url": "https://example.com/1", "content": "S1", "parsed_url": ["https","example.com","/1"]},
				{"title": "R2", "url": "https://example.com/2", "content": "S2", "parsed_url": ["https","example.com","/2"]},
				{"title": "R3", "url": "https://example.com/3", "content": "S3", "parsed_url": ["https","example.com","/3"]},
				{"title": "R4", "url": "https://example.com/4", "content": "S4", "parsed_url": ["https","example.com","/4"]},
				{"title": "R5", "url": "https://example.com/5", "content": "S5", "parsed_url": ["https","example.com","/5"]}
			]
		}`))
	}))
	defer server.Close()

	client := NewSearXNGClient(server.URL)
	client.httpClient.Timeout = 5 * time.Second

	results, err := client.Query("test", SearXNGOptions{Limit: 2})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSearchResponse_JSONStructure(t *testing.T) {
	resp := &SearchResponse{
		Query:      "test",
		Engine:     "searxng",
		Locale:     "auto",
		Total:      1,
		SearchedAt: time.Now(),
		Results: []SearchResult{
			{Rank: 1, Title: "T", URL: "https://u.com", Snippet: "S", Source: "u.com", Engines: []string{"google"}},
		},
	}

	j := resp.RenderJSON()

	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(j), &parsed))
	assert.Equal(t, true, parsed["ok"])

	result := parsed["result"].(map[string]interface{})
	assert.Equal(t, "test", result["query"])
	assert.Equal(t, float64(1), result["total"])

	results := result["results"].([]interface{})
	assert.Len(t, results, 1)

	firstResult := results[0].(map[string]interface{})
	assert.Equal(t, float64(1), firstResult["rank"])
	assert.Equal(t, "T", firstResult["title"])
}
