package ai

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
	"mini-store-go/backend/internal/dto"
)

func (s *Service) generateTurn(ctx context.Context, messages []*schema.Message, emit func(dto.StreamChunk) error) (*schema.Message, error) {
	if emit == nil {
		return s.model.Generate(ctx, messages)
	}
	if err := emit(dto.StreamChunk{Type: "thinking", Content: "正在思考..."}); err != nil {
		return nil, err
	}
	reader, err := s.model.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, errors.New("model returned an empty stream")
	}
	var once sync.Once
	closeReader := func() { once.Do(reader.Close) }
	defer closeReader()
	type received struct {
		message *schema.Message
		err     error
	}
	results := make(chan received)
	go func() {
		defer closeReader()
		for {
			message, err := reader.Recv()
			select {
			case results <- received{message, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	var chunks []*schema.Message
	filter := &contentFilter{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case item := <-results:
			if errors.Is(item.err, io.EOF) {
				if len(chunks) == 0 {
					return nil, errors.New("empty model stream")
				}
				if delta := filter.flush(); delta != "" {
					if err := emit(dto.StreamChunk{Type: "partial", Content: delta}); err != nil {
						return nil, err
					}
				}
				return schema.ConcatMessages(chunks)
			}
			if item.err != nil {
				return nil, item.err
			}
			if item.message == nil {
				continue
			}
			chunks = append(chunks, item.message)
			if len(item.message.ToolCalls) > 0 && !filter.suppressed {
				filter.suppressed = true
				if err := emit(dto.StreamChunk{Type: "tool_call", Content: "正在查询商品信息..."}); err != nil {
					return nil, err
				}
			}
			wasSuppressed := filter.suppressed
			delta := filter.push(item.message.Content)
			if !wasSuppressed && filter.suppressed {
				if err := emit(dto.StreamChunk{Type: "tool_call", Content: "正在查询商品信息..."}); err != nil {
					return nil, err
				}
			}
			if delta != "" {
				if err := emit(dto.StreamChunk{Type: "partial", Content: delta}); err != nil {
					return nil, err
				}
			}
		}
	}
}

// Hold incomplete control tags across chunk boundaries; never expose tool arguments.
type contentFilter struct {
	pending              string
	thinking, suppressed bool
}

func (f *contentFilter) push(content string) string {
	if f.suppressed {
		return ""
	}
	input := f.pending + content
	f.pending = ""
	var out strings.Builder
	tags := []string{"<think>", "<tool_call>", "<invoke", "]<]minimax[>["}
	for input != "" {
		if f.thinking {
			i := strings.Index(input, "</think>")
			if i < 0 {
				for n := min(len(input), len("</think>")-1); n > 0; n-- {
					if strings.HasPrefix("</think>", input[len(input)-n:]) {
						f.pending = input[len(input)-n:]
						break
					}
				}
				break
			}
			input = input[i+len("</think>"):]
			f.thinking = false
			continue
		}
		held := false
		for _, tag := range tags {
			if strings.HasPrefix(input, tag) {
				input = input[len(tag):]
				switch tag {
				case "<think>":
					f.thinking = true
				case "<tool_call>", "<invoke":
					f.suppressed = true
					return out.String()
				}
				held = true
				break
			}
			if strings.HasPrefix(tag, input) {
				f.pending = input
				return out.String()
			}
		}
		if held {
			continue
		}
		out.WriteByte(input[0])
		input = input[1:]
	}
	return out.String()
}
func (f *contentFilter) flush() string {
	if f.thinking || f.suppressed {
		return ""
	}
	value := f.pending
	f.pending = ""
	return value
}
