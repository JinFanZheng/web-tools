package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
)

const tavilySearchURL = "https://api.tavily.com/search"

// TavilyClient wraps the Tavily REST Search API.
type TavilyClient struct {
	apiKey     string
	httpClient *http.Client
}

// tavilyRequest is the JSON body sent to the Tavily /search endpoint.
type tavilyRequest struct {
	APIKey     string `json:"api_key"`
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

// tavilyResult is a single result from the Tavily API response.
type tavilyResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// tavilyResponse is the JSON response from Tavily /search.
type tavilyResponse struct {
	Results []tavilyResult `json:"results"`
}

// NewTavilyClient creates a new Tavily API client.
func NewTavilyClient(apiKey string) *TavilyClient {
	return &TavilyClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(config.DefaultTimeout) * time.Second,
		},
	}
}

// Query sends a search query to Tavily and returns normalised SearchResults.
func (c *TavilyClient) Query(query string, limit int) ([]SearchResult, error) {
	reqBody := tavilyRequest{
		APIKey:     c.apiKey,
		Query:      query,
		MaxResults: limit,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"Tavily request build failed",
			err.Error(),
			map[string]string{"query": query},
			nil,
		)
	}

	req, err := http.NewRequest("POST", tavilySearchURL, bytes.NewReader(payload))
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"Tavily request build failed",
			err.Error(),
			map[string]string{"url": tavilySearchURL},
			nil,
		)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"Tavily search request failed",
			err.Error(),
			map[string]string{"url": tavilySearchURL, "timeout": c.httpClient.Timeout.String()},
			[]string{"check network connectivity", "verify TAVILY_API_KEY is valid"},
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"failed to read Tavily response body",
			err.Error(),
			map[string]string{"url": tavilySearchURL},
			nil,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, apperrors.NewEngineError(
			"Tavily returned non-200 status",
			fmt.Sprintf("HTTP %d, body: %s", resp.StatusCode, string(body)),
			map[string]string{"url": tavilySearchURL, "status_code": fmt.Sprintf("%d", resp.StatusCode)},
			[]string{"verify TAVILY_API_KEY is valid", "check Tavily API status at https://status.tavily.com"},
		)
	}

	var tr tavilyResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, apperrors.NewExtractError(
			"Tavily response parse failed",
			err.Error(),
			map[string]string{"url": tavilySearchURL, "body_length": fmt.Sprintf("%d", len(body))},
			[]string{"json.Unmarshal"},
			[]string{"check Tavily API response format"},
		)
	}

	results := make([]SearchResult, 0, len(tr.Results))
	for i, r := range tr.Results {
		source := "unknown"
		if u, err := url.Parse(r.URL); err == nil && u.Hostname() != "" {
			source = u.Hostname()
		}
		results = append(results, SearchResult{
			Rank:    i + 1,
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Source:  source,
			Engines: []string{"tavily"},
		})
	}

	return results, nil
}
