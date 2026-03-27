package webreader

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
	"github.com/vanzheng/web-tools/internal/reader"
)

func Cmd() *cobra.Command {
	var (
		flagJSON    bool
		flagOutput  string
		flagExtract string
		flagMaxWord int
		flagTimeout int
		flagNoCache bool
		flagBrowser bool
		flagSession string
		flagUA      string
		flagFormat  string
	)

	cmd := &cobra.Command{
		Use:   "web-reader <input>",
		Short: "Extract readable content from URL or local file",
		Long: `Fetch a URL or read a local file, extract the main content, and output as Markdown.
Supports web pages, PDFs, Word, PowerPoint, Excel, and text files.`,
		Example: `  web-tools web-reader https://example.com/article
  web-tools web-reader https://example.com/article --max-words 100
  web-tools web-reader https://example.com/article --json
  web-tools web-reader https://spa-site.com/page --browser
  web-tools web-reader https://internal.company.com/doc --session work
  web-tools web-reader ./report.pdf
  web-tools web-reader ./slides.pptx -o /tmp/slides.md
  web-tools web-reader https://example.com/article --no-cache --timeout 30`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			run(args[0], flagJSON, flagOutput, flagExtract, flagMaxWord, flagTimeout, flagNoCache, flagBrowser, flagSession, flagUA, flagFormat)
		},
	}

	cmd.Flags().BoolVar(&flagJSON, "json", false, "JSON structured output")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output to file")
	cmd.Flags().StringVar(&flagExtract, "extract", "main", "Extract mode: main (body) / full (full page)")
	cmd.Flags().IntVar(&flagMaxWord, "max-words", 0, "Limit output word count (0 = unlimited)")
	cmd.Flags().IntVar(&flagTimeout, "timeout", 15, "Request timeout in seconds")
	cmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "Ignore cache, force refresh")
	cmd.Flags().BoolVar(&flagBrowser, "browser", false, "Force browser rendering via agent-browser")
	cmd.Flags().StringVar(&flagSession, "session", "", "agent-browser session name for login state")
	cmd.Flags().StringVar(&flagUA, "user-agent", "", "Custom User-Agent")
	cmd.Flags().StringVar(&flagFormat, "format", "markdown", "Output format: markdown / text / html")

	return cmd
}
// PipelineResult is the final output structure combining fetch + extract info.
type PipelineResult struct {
	Source        string            `json:"source"`
	URL           string            `json:"url,omitempty"`
	Title         string            `json:"title"`
	Content       string            `json:"content"`
	TextContent   string            `json:"text_content,omitempty"`
	HTML          string            `json:"html,omitempty"`
	Format        string            `json:"format"`
	FetchedAt     time.Time         `json:"fetched_at"`
	WordCount     int               `json:"word_count"`
	ContentType   string            `json:"content_type"`
	ExtractMode   string            `json:"extract_mode"`
	Language      string            `json:"language,omitempty"`
	PublishedTime *time.Time        `json:"published_time,omitempty"`
	ModifiedTime  *time.Time        `json:"modified_time,omitempty"`
	SiteName      string            `json:"site_name,omitempty"`
	Image         string            `json:"image,omitempty"`
	NeedsFallback bool              `json:"needs_fallback,omitempty"`
	CacheHit      bool              `json:"cache_hit,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func run(rawInput string, flagJSON bool, flagOutput string, flagExtract string, flagMaxWord int, flagTimeout int, flagNoCache bool, flagBrowser bool, flagSession string, flagUA string, flagFormat string) {
	// 1. Parse input
	input, err := reader.ParseInput(rawInput)
	if err != nil {
		apperrors.HandleError(err)
	}
	if input == nil {
		apperrors.HandleError(apperrors.NewInputError(
			"cannot recognize input",
			fmt.Sprintf("%q is not a valid URL or file path", rawInput),
			[]string{"provide an http:// or https:// URL", "provide a local file path"},
		))
	}

	// 2. Load config with flag overrides
	cfg := config.DefaultConfig()
	if flagTimeout > 0 {
		cfg.Reader.DefaultTimeout = flagTimeout
	}

	// 3. Initialize cache (URL inputs only)
	var cache *reader.Cache
	if input.Type == reader.InputURL {
		cache, _ = reader.NewCache(cfg.Reader.CacheDir, cfg.Reader.CacheTTL)
	}

	// 4. Handle file inputs
	if input.Type == reader.InputFile {
		result, err := handleFileInput(input, cfg, flagFormat)
		if err != nil {
			apperrors.HandleError(err)
		}
		outputResult(result, flagJSON, flagOutput, flagMaxWord)
		return
	}

	// 5. Handle URL inputs
	result, err := handleURLInput(input, cfg, flagUA, cache, flagNoCache, flagBrowser, flagSession)
	if err != nil {
		apperrors.HandleError(err)
	}

	// 6. Check extraction quality
	if result.NeedsFallback {
		fmt.Fprintln(os.Stderr, "[WARN] extracted content seems sparse, try --browser for JS-rendered pages")
	}

	// 7. Output
	outputResult(result, flagJSON, flagOutput, flagMaxWord)
}

// isHTTPStatusError checks if the error is an HTTP 4xx/5xx that should NOT trigger browser fallback.
// Browsers can't help with 403 (bot blocking), 404 (not found), etc.
func isHTTPStatusError(err error) bool {
	var appErr *apperrors.AppError
	if !apperrors.As(err, &appErr) {
		return false
	}
	// network errors with 4xx status or "unreachable" category (404) are pointless for browser
	return appErr.Category == "network" || appErr.Category == "unreachable"
}

func handleURLInput(input *reader.Input, cfg config.Config, customUA string, cache *reader.Cache, noCache bool, useBrowser bool, session string) (*PipelineResult, error) {
	// Check cache first
	if cache != nil && !noCache && !useBrowser {
		entry, content, hit := cache.Get(input.URL.String())
		if hit {
			fmt.Fprintln(os.Stderr, "[CACHE HIT] "+input.URL.String())
			return &PipelineResult{
				Source:      entry.URL,
				Title:       "",
				Content:     content,
				FetchedAt:   entry.CachedAt,
				WordCount:   entry.WordCount,
				ContentType: entry.ContentType,
				ExtractMode: "cached",
				CacheHit:    true,
			}, nil
		}
	}

	// --browser mode: use agent-browser directly
	if useBrowser {
		return handleBrowserInput(input, cfg, session)
	}

	// Default: fetch + extract
	fetcher := reader.NewFetcher(cfg.Reader)
	if customUA != "" {
		fetcher.SetUserAgent(customUA)
	}

	fetchResult, err := fetcher.Fetch(input.URL.String())
	if err != nil {
		// HTTP 4xx/5xx and 404: don't waste time with browser fallback
		if isHTTPStatusError(err) {
			return nil, err
		}
		// Network errors (timeout, DNS, connection refused): try browser
		if cfg.Reader.BrowserFallback {
			fmt.Fprintf(os.Stderr, "[WARN] HTTP fetch failed (%v), trying browser fallback\n", err)
			return handleBrowserInput(input, cfg, session)
		}
		return nil, err
	}
	defer fetchResult.Body.Close()

	extractor := reader.NewExtractor(cfg.Reader)
	extractResult, err := extractor.Extract(fetchResult.Body, input.URL)
	if err != nil {
		// Extraction failure: browser can help with JS-rendered pages
		if cfg.Reader.BrowserFallback {
			fmt.Fprintf(os.Stderr, "[WARN] extraction failed (%v), trying browser fallback\n", err)
			return handleBrowserInput(input, cfg, session)
		}
		return nil, err
	}

	contentType := reader.GuessContentType(input.URL.String(), extractResult.SiteName, extractResult.Metadata)
	wordCount := len(strings.Fields(extractResult.TextContent))

	result := &PipelineResult{
		Source:        input.URL.String(),
		URL:           fetchResult.URL,
		Title:         extractResult.Title,
		Content:       extractResult.Content,
		TextContent:   extractResult.TextContent,
		HTML:          extractResult.HTML,
		Format:        flagFormatFromContentType(contentType),
		FetchedAt:     time.Now(),
		WordCount:     wordCount,
		ContentType:   contentType,
		ExtractMode:   "readability",
		Language:      extractResult.Language,
		PublishedTime: extractResult.PublishedTime,
		ModifiedTime:  extractResult.ModifiedTime,
		SiteName:      extractResult.SiteName,
		Image:         extractResult.Image,
		Metadata:      extractResult.Metadata,
		NeedsFallback: extractor.NeedsFallback(extractResult),
	}

	if cache != nil {
		cacheEntry := &reader.CacheEntry{
			URL:         input.URL.String(),
			CachedAt:    time.Now(),
			WordCount:   wordCount,
			HTTPStatus:  fetchResult.StatusCode,
			ContentType: fetchResult.ContentType,
		}
		if err := cache.Set(input.URL.String(), cacheEntry, extractResult.Content); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] cache write failed: %v\n", err)
		}
	}

	return result, nil
}

func handleBrowserInput(input *reader.Input, cfg config.Config, session string) (*PipelineResult, error) {
	browser := reader.NewBrowserFallback(cfg.Reader)

	title, content, err := browser.Extract(input.URL.String(), session)
	if err != nil {
		return nil, err
	}
	defer browser.Close(session)

	wordCount := len(strings.Fields(content))

	return &PipelineResult{
		Source:      input.URL.String(),
		Title:       title,
		Content:     content,
		FetchedAt:   time.Now(),
		WordCount:   wordCount,
		ContentType: reader.GuessContentType(input.URL.String(), "", nil),
		ExtractMode: "browser",
		Metadata: map[string]string{
			"engine": "agent-browser",
		},
	}, nil
}

func handleFileInput(input *reader.Input, cfg config.Config, flagFormat string) (*PipelineResult, error) {
	data, err := os.ReadFile(input.FilePath)
	if err != nil {
		return nil, apperrors.NewInputError(
			"cannot read file",
			err.Error(),
			[]string{"check file path", "check read permissions"},
		)
	}

	content := string(data)

	// Text files: return directly
	if !input.NeedsConversion() {
		return &PipelineResult{
			Source:    input.FilePath,
			Title:     input.FilePath,
			Content:   content,
			FetchedAt: time.Now(),
			WordCount: len(strings.Fields(content)),
			Format:    flagFormat,
			Metadata: map[string]string{
				"source_type": "file",
				"extension":   input.Extension(),
			},
		}, nil
	}

	// Binary files: use markitdown converter
	converter := reader.NewConverter(cfg.Reader)
	if !converter.Available() {
		return nil, apperrors.NewEngineError(
			"markitdown not found",
			fmt.Sprintf("file type %s needs markitdown, but %q not found", input.Extension(), cfg.Reader.MarkitdownPath),
			map[string]string{"file": input.FilePath, "extension": input.Extension()},
			[]string{"install markitdown: pip install markitdown", "or: pipx install markitdown"},
		)
	}

	converted, err := converter.Convert(input.FilePath)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Source:    input.FilePath,
		Title:     input.FilePath,
		Content:   converted,
		FetchedAt: time.Now(),
		WordCount: len(strings.Fields(converted)),
		Format:    flagFormat,
		ExtractMode: "markitdown",
		Metadata: map[string]string{
			"source_type": "file",
			"extension":   input.Extension(),
			"converter":   "markitdown",
		},
	}, nil
}

func outputResult(result *PipelineResult, flagJSON bool, flagOutput string, flagMaxWord int) {
	if flagMaxWord > 0 {
		words := strings.Fields(result.Content)
		if len(words) > flagMaxWord {
			result.Content = strings.Join(words[:flagMaxWord], " ") + "\n\n... (truncated)"
			result.WordCount = flagMaxWord
		}
	}

	var output string
	if flagJSON {
		output = result.RenderJSON()
	} else {
		output = result.RenderMarkdown()
	}

	if flagOutput != "" {
		if err := os.WriteFile(flagOutput, []byte(output), 0644); err != nil {
			apperrors.HandleError(apperrors.NewInputError(
				"cannot write to output file",
				err.Error(),
				[]string{"check output path write permissions"},
			))
		}
	} else {
		fmt.Println(output)
	}
}

func flagFormatFromContentType(ct string) string {
	switch ct {
	case "documentation", "forum", "article":
		return "markdown"
	case "video", "social":
		return "text"
	default:
		return "markdown"
	}
}

func (r *PipelineResult) RenderMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<!-- source: %s -->\n", r.Source))
	if r.URL != "" && r.URL != r.Source {
		sb.WriteString(fmt.Sprintf("<!-- url: %s -->\n", r.URL))
	}
	sb.WriteString(fmt.Sprintf("<!-- title: %s -->\n", r.Title))
	sb.WriteString(fmt.Sprintf("<!-- fetched: %s -->\n", r.FetchedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("<!-- words: %d -->\n", r.WordCount))
	if r.ContentType != "" {
		sb.WriteString(fmt.Sprintf("<!-- type: %s -->\n", r.ContentType))
	}
	if r.ExtractMode != "" {
		sb.WriteString(fmt.Sprintf("<!-- extract_mode: %s -->\n", r.ExtractMode))
	}
	if r.PublishedTime != nil {
		sb.WriteString(fmt.Sprintf("<!-- published: %s -->\n", r.PublishedTime.Format(time.RFC3339)))
	}
	if r.ModifiedTime != nil {
		sb.WriteString(fmt.Sprintf("<!-- modified: %s -->\n", r.ModifiedTime.Format(time.RFC3339)))
	}
	if r.Language != "" {
		sb.WriteString(fmt.Sprintf("<!-- language: %s -->\n", r.Language))
	}
	if r.SiteName != "" {
		sb.WriteString(fmt.Sprintf("<!-- site: %s -->\n", r.SiteName))
	}
	if r.CacheHit {
		sb.WriteString("<!-- cache: hit -->\n")
	}
	sb.WriteString("\n")

	if r.Title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", r.Title))
	}

	if len(r.Metadata) > 0 {
		for k, v := range r.Metadata {
			if v != "" {
				sb.WriteString(fmt.Sprintf("> **%s:** %s\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(r.Content)
	sb.WriteString("\n")

	return sb.String()
}

func (r *PipelineResult) RenderJSON() string {
	type jsonOutput struct {
		OK     bool            `json:"ok"`
		Result *PipelineResult `json:"result"`
	}
	resp := jsonOutput{OK: true, Result: r}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		data, _ = json.Marshal(resp)
	}
	return string(data)
}
