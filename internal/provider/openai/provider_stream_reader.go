package openai

import (
	"io"

	"github.com/Jayleonc/ai-gateway/internal/provider"
)

type providerStreamReader struct {
	r *StreamReader
}

func (psr *providerStreamReader) Recv() (*provider.StreamEvent, error) {
	delta, err := psr.r.Next()
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, err
	}
	return &provider.StreamEvent{
		Delta: &provider.Message{
			Role:    "assistant",
			Content: delta.Text,
		},
	}, nil
}

func (psr *providerStreamReader) Close() error {
	return psr.r.Close()
}

var _ provider.StreamReader = (*providerStreamReader)(nil)
