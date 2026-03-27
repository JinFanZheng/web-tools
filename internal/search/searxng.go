package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
)

// SearXNGClient wraps the local SearXNG JSON API.
type SearXNGClient struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// SearXNGResult represents a single result from SearXNG API.
// ParsedURL is kept as raw JSON to handle both array-of-strings (SearXNG default)
// and array-of-objects formats across versions.
type SearXNGResult struct {
	Title         string          `json:"title"`
	URL           string          `json:"url"`
	Content       string          `json:"content"`    // snippet
	Engines       []string        `json:"engines"`
	ParsedURL     json.RawMessage `json:"parsed_url"`
	PublishedDate *string         `json:"publishedDate"`
}

// SearXNGOptions holds query parameters for the SearXNG API.
type SearXNGOptions struct {
	Limit     int
	Locale    string // "auto" / "zh-CN" / "en-US"
	Category  string // "general" / "images" / "news" / "videos" / "files"
	TimeRange string // "" / "day" / "week" / "month" / "year"
}

// searxngResponse is the raw JSON response from SearXNG.
type searxngResponse struct {
	Results   []SearXNGResult `json:"results"`
	NumberOf int              `json:"number_of_results"`
}

// NewSearXNGClient creates a new client for the local SearXNG instance.
func NewSearXNGClient(baseURL string) *SearXNGClient {
	return &SearXNGClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(config.DefaultTimeout) * time.Second,
		},
		userAgent: "web-tools/1.0 (agent CLI)",
	}
}

// HealthCheck verifies that the SearXNG instance is reachable.
func (c *SearXNGClient) HealthCheck() error {
	client := &http.Client{Timeout: config.HealthCheckTimeout}
	req, err := http.NewRequest("HEAD", c.baseURL, nil)
	if err != nil {
		return apperrors.NewEngineError(
			"SearXNG health check request failed",
			err.Error(),
			map[string]string{"searxng_url": c.baseURL},
			[]string{
				fmt.Sprintf("check SearXNG: curl -s -o /dev/null -w %%{http_code} %s", c.baseURL),
				"start SearXNG: cd docker && docker compose up -d",
			},
		)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return apperrors.NewEngineError(
			"SearXNG unreachable",
			err.Error(),
			map[string]string{"searxng_url": c.baseURL, "timeout": config.HealthCheckTimeout.String()},
			[]string{
				"start SearXNG: cd docker && docker compose up -d",
				"check Docker: docker ps",
				fmt.Sprintf("confirm port config: %s", c.baseURL),
			},
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return apperrors.NewEngineError(
			"SearXNG service error",
			fmt.Sprintf("HTTP %d", resp.StatusCode),
			map[string]string{"searxng_url": c.baseURL, "status_code": fmt.Sprintf("%d", resp.StatusCode)},
			[]string{"restart SearXNG: cd docker && docker compose restart", "logs: cd docker && docker compose logs"},
		)
	}

	return nil
}

// Query sends a search query to SearXNG and returns parsed results.
func (c *SearXNGClient) Query(query string, opts SearXNGOptions) ([]SearXNGResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("categories", opts.Category)
	if opts.Locale != "" && opts.Locale != "auto" {
		params.Set("language", opts.Locale)
	}
	if opts.TimeRange != "" && opts.TimeRange != "any" {
		params.Set("time_range", opts.TimeRange)
	}

	reqURL := c.baseURL + "/search?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"search request build failed",
			err.Error(),
			map[string]string{"url": reqURL},
			nil,
		)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"SearXNG search request failed",
			err.Error(),
			map[string]string{"url": reqURL, "timeout": c.httpClient.Timeout.String()},
			[]string{"check network", "confirm SearXNG container is running"},
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"failed to read SearXNG response body",
			err.Error(),
			map[string]string{"url": reqURL},
			nil,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, apperrors.NewEngineError(
			"SearXNG returned non-200 status",
			fmt.Sprintf("HTTP %d, body: %s", resp.StatusCode, string(body)),
			map[string]string{"url": reqURL, "status_code": fmt.Sprintf("%d", resp.StatusCode)},
			[]string{"SearXNG logs: cd docker && docker compose logs", "try a simple query"},
		)
	}

	var sxr searxngResponse
	if err := json.Unmarshal(body, &sxr); err != nil {
		return nil, apperrors.NewExtractError(
			"SearXNG response parse failed",
			err.Error(),
			map[string]string{"url": reqURL, "body_length": fmt.Sprintf("%d", len(body))},
			[]string{"json.Unmarshal"},
			[]string{"check SearXNG version compatibility", "check SearXNG logs"},
		)
	}

	// Apply limit
	if opts.Limit > 0 && len(sxr.Results) > opts.Limit {
		sxr.Results = sxr.Results[:opts.Limit]
	}

	return sxr.Results, nil
}

// ExtractSource gets the source domain from a SearXNG result.
// Tries parsed_url first (handles both string-array and object-array formats),
// then falls back to parsing the URL directly.
func ExtractSource(r SearXNGResult) string {
	// Try to extract from parsed_url (format: ["https", "host", "/path", ...])
	if len(r.ParsedURL) > 0 {
		var strArr []string
		if err := json.Unmarshal(r.ParsedURL, &strArr); err == nil && len(strArr) >= 2 {
			host := strArr[1]
			if host != "" {
				return host
			}
		}
		// Try object-array format: [{"scheme":"https","netloc":"host"},...]
		var objArr []struct {
			Netloc string `json:"netloc"`
		}
		if err := json.Unmarshal(r.ParsedURL, &objArr); err == nil && len(objArr) > 0 && objArr[0].Netloc != "" {
			return objArr[0].Netloc
		}
	}
	// Fallback: parse URL
	if u, err := url.Parse(r.URL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "unknown"
}
