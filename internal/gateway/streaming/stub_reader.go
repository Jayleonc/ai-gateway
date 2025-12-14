package streaming

import (
	"io"
	"time"
)

// StubStreamReader 用于演示的 stub stream reader
// 实际应用中会从 provider 的 response body 读取真实 SSE 流
type StubStreamReader struct {
	chunks []*Delta
	index  int
	delay  time.Duration
}

// NewStubStreamReader 创建 stub reader
func NewStubStreamReader(chunks []*Delta, delay time.Duration) *StubStreamReader {
	return &StubStreamReader{
		chunks: chunks,
		index:  0,
		delay:  delay,
	}
}

// Next 返回下一个 chunk，或 io.EOF 表示流结束
func (sr *StubStreamReader) Next() (*Delta, error) {
	if sr.index >= len(sr.chunks) {
		return nil, io.EOF
	}

	if sr.delay > 0 {
		time.Sleep(sr.delay)
	}

	delta := sr.chunks[sr.index]
	sr.index++
	return delta, nil
}

// Close 关闭 reader
func (sr *StubStreamReader) Close() error {
	return nil
}
