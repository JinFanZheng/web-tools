package search

import (
	"encoding/json"
	"fmt"
	"github.com/vanzheng/web-tools/internal/config"
	"strings"
	"time"
)

// Search is the main entry point for web search.
type Search struct {
	client       *SearXNGClient
	tavilyClient *TavilyClient
	config       config.SearchConfig
	engine       string
}

// SearchOptions holds user-facing search options.
type SearchOptions struct {
	Limit     int
	Locale    string // "auto" / "zh-CN" / "en-US"
	Category  string // "general" / "images" / "news" / "videos" / "files"
	TimeRange string // "" / "any" / "day" / "week" / "month" / "year"
}

// SearchResult is a single normalized search result.
type SearchResult struct {
	Rank          int      `json:"rank"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Snippet       string   `json:"snippet"`
	Source        string   `json:"source"`
	Engines       []string `json:"engines"`
	PublishedDate string   `json:"published_date,omitempty"`
}

// SearchResponse is the final output structure.
type SearchResponse struct {
	Query      string         `json:"query"`
	Engine     string         `json:"engine"`
	Locale     string         `json:"locale"`
	Total      int            `json:"total"`
	Results    []SearchResult `json:"results"`
	SearchedAt time.Time      `json:"searched_at"`
}

// NewSearch creates a new Search instance.
// The engine parameter selects the search backend ("searxng" or "tavily").
func NewSearch(cfg config.SearchConfig, engine string) *Search {
	s := &Search{
		client: NewSearXNGClient(cfg.SearXNGURL),
		config: cfg,
		engine: engine,
	}
	if cfg.TavilyAPIKey != "" {
		s.tavilyClient = NewTavilyClient(cfg.TavilyAPIKey)
	}
	return s
}

// Do performs a search: health check + query + normalize results.
func (s *Search) Do(query string, opts SearchOptions) (*SearchResponse, error) {
	// Apply defaults
	if opts.Limit <= 0 {
		opts.Limit = s.config.DefaultLimit
	}
	if opts.Category == "" {
		opts.Category = "general"
	}

	locale := opts.Locale
	if locale == "" {
		locale = "auto"
	}

	if s.engine == "tavily" {
		return s.doTavily(query, opts, locale)
	}
	return s.doSearXNG(query, opts, locale)
}

func (s *Search) doTavily(query string, opts SearchOptions, locale string) (*SearchResponse, error) {
	if s.tavilyClient == nil {
		return nil, fmt.Errorf("TAVILY_API_KEY is not set; required for --engine tavily")
	}

	// Map category to Tavily topic (general, news, finance are supported).
	topic := opts.Category
	if topic == "" {
		topic = "general"
	}

	// Map time range: pass through if set and not "any".
	timeRange := ""
	if opts.TimeRange != "" && opts.TimeRange != "any" {
		timeRange = opts.TimeRange
	}

	results, err := s.tavilyClient.Query(query, opts.Limit, topic, timeRange)
	if err != nil {
		return nil, err
	}

	return &SearchResponse{
		Query:      query,
		Engine:     "tavily",
		Locale:     locale,
		Total:      len(results),
		Results:    results,
		SearchedAt: time.Now(),
	}, nil
}

func (s *Search) doSearXNG(query string, opts SearchOptions, locale string) (*SearchResponse, error) {
	// Health check first
	if err := s.client.HealthCheck(); err != nil {
		return nil, err
	}

	// Map to SearXNG options
	sxOpts := SearXNGOptions{
		Limit:     opts.Limit,
		Locale:    opts.Locale,
		Category:  opts.Category,
		TimeRange: opts.TimeRange,
	}

	// Execute query
	rawResults, err := s.client.Query(query, sxOpts)
	if err != nil {
		return nil, err
	}

	// Normalize results
	results := make([]SearchResult, 0, len(rawResults))
	for i, r := range rawResults {
		publishedDate := ""
		if r.PublishedDate != nil {
			publishedDate = *r.PublishedDate
		}
		results = append(results, SearchResult{
			Rank:          i + 1,
			Title:         r.Title,
			URL:           r.URL,
			Snippet:       r.Content,
			Source:        ExtractSource(r),
			Engines:       r.Engines,
			PublishedDate: publishedDate,
		})
	}

	return &SearchResponse{
		Query:      query,
		Engine:     "searxng",
		Locale:     locale,
		Total:      len(results),
		Results:    results,
		SearchedAt: time.Now(),
	}, nil
}

// RenderMarkdown outputs the search response as Markdown.
func (r *SearchResponse) RenderMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Search: \"%s\"\n", r.Query))
	sb.WriteString(fmt.Sprintf("> Engine: %s | Locale: %s | Results: %d | %s\n\n",
		r.Engine, r.Locale, r.Total, r.SearchedAt.Format(time.RFC3339)))

	for _, result := range r.Results {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", result.Rank, result.Title))
		sb.WriteString(fmt.Sprintf("**Source:** %s\n", result.Source))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n", result.URL))
		if result.PublishedDate != "" {
			sb.WriteString(fmt.Sprintf("**Published:** %s\n", result.PublishedDate))
		}
		if result.Snippet != "" {
			sb.WriteString(fmt.Sprintf("**Snippet:** %s\n", result.Snippet))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderJSON outputs the search response as JSON.
func (r *SearchResponse) RenderJSON() string {
	type jsonOutput struct {
		OK     bool           `json:"ok"`
		Result *SearchResponse `json:"result"`
	}
	resp := jsonOutput{OK: true, Result: r}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		data, _ = json.Marshal(resp)
	}
	return string(data)
}
