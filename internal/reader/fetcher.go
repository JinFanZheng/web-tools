package reader

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vanzheng/web-tools/internal/config"
	apperrors "github.com/vanzheng/web-tools/internal/errors"
)

// FetchResult holds the HTTP response data.
type FetchResult struct {
	URL         string
	StatusCode int
	ContentType string
	Body        io.ReadCloser
	Elapsed     time.Duration
}

// Fetcher handles HTTP requests for web pages.
type Fetcher struct {
	client    *http.Client
	userAgent string
}

// NewFetcher creates a new Fetcher with the given configuration.
func NewFetcher(cfg config.ReaderConfig) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: time.Duration(cfg.DefaultTimeout) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= config.MaxRedirects {
					return fmt.Errorf("too many redirects (>%d)", config.MaxRedirects)
				}
				return nil
			},
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// Fetch performs an HTTP GET request for the given URL.
func (f *Fetcher) Fetch(rawURL string) (*FetchResult, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, apperrors.NewInputError(
			"无效的 URL",
			err.Error(),
			[]string{"检查 URL 格式是否正确"},
		)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, apperrors.NewInputError(
			"不支持的 URL 协议",
			fmt.Sprintf("仅支持 http/https，当前: %s", parsedURL.Scheme),
			[]string{"使用 http:// 或 https:// 开头的 URL"},
		)
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, apperrors.NewNetworkError(
			"请求构建失败",
			err.Error(),
			map[string]string{"url": rawURL},
			nil,
		)
	}
	req.Header.Set("User-Agent", f.userAgent)

	start := time.Now()
	resp, err := f.client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return nil, apperrors.NewNetworkError(
			"HTTP 请求失败",
			err.Error(),
			map[string]string{"url": rawURL, "elapsed": elapsed.String()},
			[]string{
				"检查网络连接",
				fmt.Sprintf("当前超时设置: %s", f.client.Timeout),
			},
		)
	}

	if resp.StatusCode == 404 {
		resp.Body.Close()
		return nil, apperrors.NewUnreachableError(
			"页面不存在",
			fmt.Sprintf("HTTP %d", resp.StatusCode),
			map[string]string{"url": rawURL},
			[]string{"确认 URL 是否正确"},
		)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, apperrors.NewNetworkError(
			"HTTP 请求返回错误状态码",
			fmt.Sprintf("HTTP %d", resp.StatusCode),
			map[string]string{"url": rawURL, "status_code": fmt.Sprintf("%d", resp.StatusCode)},
			[]string{"确认 URL 是否需要登录或权限"},
		)
	}

	return &FetchResult{
		URL:         resp.Request.URL.String(),
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        resp.Body,
		Elapsed:     elapsed,
	}, nil
}

// SetUserAgent overrides the default User-Agent.
func (f *Fetcher) SetUserAgent(ua string) {
	if ua != "" {
		f.userAgent = ua
	}
}
