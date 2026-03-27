package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vanzheng/web-tools/cmd/web-reader"
	"github.com/vanzheng/web-tools/cmd/web-search"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "web-tools",
		Short:   "Local-first web tools for AI agents",
		Long: `web-tools provides web-search and web-reader as CLI tools, designed for AI agents to consume.

Zero cost. No API keys. No third-party dependencies.`,
		Example: `  web-tools web-search "latest AI news" --limit 3
  web-tools web-search "人工智能" --locale zh-CN --time-range week
  web-tools web-search "site:reuters.com Iran" --category news
  web-tools web-reader https://example.com/article
  web-tools web-reader https://example.com/spa-page --browser
  web-tools web-reader ./report.pdf
  web-tools web-reader ./slides.pptx -o /tmp/slides.md`,
		Version: version,
	}

	rootCmd.AddCommand(webreader.Cmd())
	rootCmd.AddCommand(websearch.Cmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
