package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_ToStderr(t *testing.T) {
	err := NewExtractError(
		"正文提取失败",
		"提取内容仅 12 字",
		map[string]string{"url": "https://example.com"},
		[]string{"readability", "agent-browser"},
		[]string{"尝试 --browser 强制使用浏览器渲染"},
	)

	output := err.ToStderr()
	assert.Contains(t, output, "[ERROR] extract: 正文提取失败")
	assert.Contains(t, output, "url: https://example.com")
	assert.Contains(t, output, "原因: 提取内容仅 12 字")
	assert.Contains(t, output, "已尝试: readability → agent-browser")
	assert.Contains(t, output, "建议: 尝试 --browser 强制使用浏览器渲染")
}

func TestAppError_ToJSON(t *testing.T) {
	err := NewEngineError(
		"SearXNG 服务不可用",
		"连接被拒绝",
		map[string]string{"address": "http://localhost:8888"},
		[]string{"docker ps | grep searxng"},
	)

	data := err.ToJSON()
	assert.Contains(t, string(data), `"ok": false`)
	assert.Contains(t, string(data), `"engine"`)
	assert.Contains(t, string(data), `"SearXNG 服务不可用"`)
	assert.Contains(t, string(data), `"连接被拒绝"`)
}

func TestAppError_ExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected int
	}{
		{"network", NewNetworkError("", "", nil, nil), 1},
		{"unreachable", NewUnreachableError("", "", nil, nil), 2},
		{"extract", NewExtractError("", "", nil, nil, nil), 3},
		{"engine", NewEngineError("", "", nil, nil), 4},
		{"input", NewInputError("", "", nil), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.ExitCode())
		})
	}
}

func TestAppError_Error(t *testing.T) {
	err := NewNetworkError("请求超时", "30 秒无响应", nil, nil)
	assert.Contains(t, err.Error(), "[network]")
	assert.Contains(t, err.Error(), "请求超时")
}
