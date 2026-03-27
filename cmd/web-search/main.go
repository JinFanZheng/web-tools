package websearch

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
	"github.com/vanzheng/web-tools/internal/search"
)

func Cmd() *cobra.Command {
	var (
		flagJSON    bool
		flagOutput  string
		flagLimit   int
		flagEngine  string
		flagLocale  string
		flagCat     string
		flagTime    string
	)

	cmd := &cobra.Command{
		Use:   "web-search <query>",
		Short: "Search the web via SearXNG or Tavily",
		Long: `Search the web using a local SearXNG instance (default) or the Tavily API.

SearXNG aggregates Google, Bing, DuckDuckGo — zero cost, no API keys, no rate limits.
Tavily is an optional cloud-based engine requiring TAVILY_API_KEY to be set.
Use --engine tavily to select Tavily. With Tavily, --category maps to topic (general/news/finance)
and --time-range supports day/week/month/year.`,
		Example: `  web-tools web-search "latest AI news"
  web-tools web-search "人工智能最新进展" --locale zh-CN --time-range week
  web-tools web-search "Tesla" --category news --time-range day --limit 10
  web-tools web-search "site:github.com go readability" --limit 3 --json
  web-tools web-search "深度学习" --locale zh-CN --limit 3 -o /tmp/results.md
  web-tools web-search "climate change 2026" --time-range year --json`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			run(args[0], flagJSON, flagOutput, flagLimit, flagEngine, flagLocale, flagCat, flagTime)
		},
	}

	cmd.Flags().BoolVar(&flagJSON, "json", false, "JSON structured output")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output to file")
	cmd.Flags().IntVarP(&flagLimit, "limit", "n", 5, "Number of results")
	cmd.Flags().StringVar(&flagEngine, "engine", "searxng", "Search engine: searxng (default) or tavily")
	cmd.Flags().StringVar(&flagLocale, "locale", "auto", "Language preference (zh-CN, en-US, auto)")
	cmd.Flags().StringVar(&flagCat, "category", "general", "Search category: general / images / news / videos / files")
	cmd.Flags().StringVar(&flagTime, "time-range", "any", "Time range: any / day / week / month / year")

	return cmd
}

func run(query string, flagJSON bool, flagOutput string, flagLimit int, flagEngine string, flagLocale string, flagCategory string, flagTimeRange string) {
	if flagEngine != "searxng" && flagEngine != "tavily" {
		apperrors.HandleError(apperrors.NewInputError(
			"unsupported engine",
			fmt.Sprintf("engine %q is not supported", flagEngine),
			[]string{"supported engines: searxng, tavily", "use --engine searxng (default)"},
		))
	}

	cfg, err := config.Load()
	if err != nil {
		apperrors.HandleError(apperrors.NewInputError(
			"failed to load configuration",
			err.Error(),
			[]string{"check config file format", "verify environment variables"},
		))
	}
	s := search.NewSearch(cfg.Search, flagEngine)

	opts := search.SearchOptions{
		Limit:     flagLimit,
		Locale:    flagLocale,
		Category:  flagCategory,
		TimeRange: flagTimeRange,
	}

	resp, err := s.Do(query, opts)
	if err != nil {
		apperrors.HandleError(err)
	}

	var output string
	if flagJSON {
		output = resp.RenderJSON()
	} else {
		output = resp.RenderMarkdown()
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
