package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	openaitypes "github.com/Jayleonc/ai-gateway/pkg/openai"
)

// Delta 表示从 OpenAI 的 chat.completion.chunk SSE 事件中提取出的最小增量。
// 该结构刻意不包含任何治理或计量语义。
type Delta struct {
	Text string
}

// StreamReader 用于读取 OpenAI 兼容的 SSE 流式响应。
// 它只负责传输层处理（HTTP + SSE + JSON 解析）。
// 它不实现任何治理逻辑。
type StreamReader struct {
	resp   *http.Response
	reader *bufio.Reader

	done bool
}

// NewStreamReader 使用给定的 http.Client 发起请求并创建 StreamReader。
func NewStreamReader(ctx context.Context, client *http.Client, req *http.Request) (*StreamReader, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if req == nil {
		return nil, coreerrors.ErrBadRequest
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()

		var apiErr openaitypes.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Error != nil && apiErr.Error.Message != "" {
			return nil, coreerrors.NewAPIError("openai_error", apiErr.Error.Message, apiErr.Error.Type)
		}
		return nil, fmt.Errorf("openai streaming: unexpected status %d", resp.StatusCode)
	}

	return &StreamReader{
		resp:   resp,
		reader: bufio.NewReader(resp.Body),
	}, nil
}

// Next 从 SSE 流中读取下一个 Delta。
// 当流结束时返回 io.EOF（包含收到 [DONE] 的情况）。
func (sr *StreamReader) Next() (*Delta, error) {
	if sr == nil || sr.reader == nil {
		return nil, io.EOF
	}
	if sr.done {
		return nil, io.EOF
	}

	for {
		payload, err := sr.readSSEDataPayload()
		if err != nil {
			return nil, err
		}
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			sr.done = true
			return nil, io.EOF
		}

		delta, err := parseChatCompletionChunkDelta(payload)
		if err != nil {
			return nil, err
		}
		return delta, nil
	}
}

func (sr *StreamReader) Close() error {
	if sr == nil || sr.resp == nil || sr.resp.Body == nil {
		return nil
	}
	return sr.resp.Body.Close()
}

// readSSEDataPayload 读取一个 SSE event，并返回该 event 内所有 "data:" 行拼接后的 payload。
// event 边界由空行分隔。
func (sr *StreamReader) readSSEDataPayload() (string, error) {
	var parts []string
	for {
		line, err := sr.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// 如果已累计到一个 event 的 payload，则返回该 payload；否则结束。
				if len(parts) > 0 {
					return strings.TrimSpace(strings.Join(parts, "\n")), nil
				}
				return "", io.EOF
			}
			return "", err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if len(parts) == 0 {
				return "", nil
			}
			return strings.TrimSpace(strings.Join(parts, "\n")), nil
		}

		// SSE 注释行
		if strings.HasPrefix(line, ":") {
			continue
		}
		// 忽略非 data 字段（event:, id:, retry:）
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		parts = append(parts, data)
	}
}

type chatCompletionChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

func parseChatCompletionChunkDelta(payload string) (*Delta, error) {
	var chunk chatCompletionChunk
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return nil, fmt.Errorf("openai streaming: decode chunk: %w", err)
	}

	var b strings.Builder
	for _, c := range chunk.Choices {
		if c.Delta.Content != "" {
			b.WriteString(c.Delta.Content)
		}
	}

	return &Delta{Text: b.String()}, nil
}
