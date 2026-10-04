package ai

import (
	"context"
	"errors"
	"github.com/cloudwego/eino/schema"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/dto"
	"strings"
	"testing"
	"time"
)

type agentStreamModel struct {
	turns         [][]*schema.Message
	history       [][]*schema.Message
	generateCalls int
	delay         time.Duration
}

func (m *agentStreamModel) Generate(context.Context, []*schema.Message) (*schema.Message, error) {
	m.generateCalls++
	return nil, errors.New("stream must not regenerate")
}
func (m *agentStreamModel) BindTools([]*schema.ToolInfo) error { return nil }
func (m *agentStreamModel) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	index := len(m.history)
	m.history = append(m.history, append([]*schema.Message(nil), messages...))
	reader, writer := schema.Pipe[*schema.Message](8)
	go func() {
		defer writer.Close()
		if index >= len(m.turns) {
			writer.Send(nil, errors.New("unexpected model turn"))
			return
		}
		for _, chunk := range m.turns[index] {
			if writer.Send(chunk, nil) {
				return
			}
		}
	}()
	return reader, nil
}

func TestStreamContinuesOriginalToolConversation(t *testing.T) {
	index := 0
	model := &agentStreamModel{turns: [][]*schema.Message{
		{
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{Index: &index, ID: "search-1", Type: "function", Function: schema.FunctionCall{Name: "search_products", Arguments: `{"query":"pho`}}}},
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{Index: &index, Function: schema.FunctionCall{Arguments: `ne"}`}}}},
		},
		{{Role: schema.Assistant, Content: "<thi"}, {Role: schema.Assistant, Content: "nk>private</think>Found "}, {Role: schema.Assistant, Content: "phones"}},
	}}
	products := &fakeProductRepository{}
	service := NewService(config.AIConfig{Enabled: true}, model, products, nil, nil)
	var events []dto.StreamChunk
	output, err := service.Stream(context.Background(), dto.ChatInput{Messages: []dto.ChatMessageInput{{Role: "user", Content: "phones"}}}, func(chunk dto.StreamChunk) error { events = append(events, chunk); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if model.generateCalls != 0 || len(model.history) != 2 {
		t.Fatalf("model calls: generate=%d stream=%d", model.generateCalls, len(model.history))
	}
	if products.lastFilter.Query != "phone" {
		t.Fatalf("fragmented arguments were lost: %#v", products.lastFilter)
	}
	history := model.history[1]
	if history[len(history)-2].Role != schema.Tool || history[len(history)-2].ToolCallID != "search-1" {
		t.Fatalf("missing original tool result: %#v", history)
	}
	var visible strings.Builder
	var toolStarted, toolFinished bool
	for _, event := range events {
		if event.Type == "partial" {
			visible.WriteString(event.Content)
		}
		if event.Type == "tool_call" {
			toolStarted = true
		}
		if event.Type == "tool_result" {
			toolFinished = true
		}
	}
	if visible.String() != "Found phones" || output.Content != "Found phones" || !toolStarted || !toolFinished {
		t.Fatalf("unexpected stream: %#v, %#v", events, output)
	}
}

func TestStreamXMLToolMarkupSplitAtEveryByte(t *testing.T) {
	content := `]<]minimax[>[<tool_call><invoke name="search_products"><query>phone</query></invoke></tool_call>`
	var chunks []*schema.Message
	for _, char := range content {
		chunks = append(chunks, &schema.Message{Role: schema.Assistant, Content: string(char)})
	}
	model := &agentStreamModel{turns: [][]*schema.Message{chunks, {{Role: schema.Assistant, Content: "Answer"}}}}
	service := NewService(config.AIConfig{Enabled: true}, model, &fakeProductRepository{}, nil, nil)
	var visible string
	_, err := service.Stream(context.Background(), dto.ChatInput{}, func(chunk dto.StreamChunk) error {
		if chunk.Type == "partial" {
			visible += chunk.Content
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if visible != "Answer" {
		t.Fatalf("leaked markup: %q", visible)
	}
}

func TestAgentTimeoutAppliesToWholeConversation(t *testing.T) {
	model := &agentStreamModel{delay: 30 * time.Millisecond, turns: [][]*schema.Message{
		{{Role: schema.Assistant, Content: `<tool_call><invoke name="search_products"><query>phone</query></invoke></tool_call>`}},
		{{Role: schema.Assistant, Content: "too late"}},
	}}
	service := NewService(config.AIConfig{Enabled: true, Timeout: 50 * time.Millisecond}, model, &fakeProductRepository{}, nil, nil)
	_, err := service.Stream(context.Background(), dto.ChatInput{}, func(dto.StreamChunk) error { return nil })
	if err == nil || len(model.history) != 1 {
		t.Fatalf("total deadline not enforced: %v, turns=%d", err, len(model.history))
	}
}

func TestStreamEmitterFailureStopsBeforeModelCall(t *testing.T) {
	model := &agentStreamModel{}
	service := NewService(config.AIConfig{Enabled: true}, model, nil, nil, nil)
	_, err := service.Stream(context.Background(), dto.ChatInput{}, func(dto.StreamChunk) error { return errors.New("client disconnected") })
	if err == nil || len(model.history) != 0 {
		t.Fatal("generation continued after client disconnected")
	}
}

type stalledStreamModel struct{ agentStreamModel }

func (m *stalledStreamModel) Stream(ctx context.Context, _ []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	reader, writer := schema.Pipe[*schema.Message](0)
	go func() { <-ctx.Done(); writer.Close() }()
	return reader, nil
}
func TestStreamDeadlineInterruptsPendingRead(t *testing.T) {
	service := NewService(config.AIConfig{Enabled: true, Timeout: 20 * time.Millisecond}, &stalledStreamModel{}, nil, nil, nil)
	completed := make(chan error, 1)
	go func() {
		_, err := service.Stream(context.Background(), dto.ChatInput{}, func(dto.StreamChunk) error { return nil })
		completed <- err
	}()
	select {
	case err := <-completed:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stalled reader ignored deadline")
	}
}
