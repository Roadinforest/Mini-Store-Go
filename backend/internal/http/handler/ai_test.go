package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	einoschema "github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/ai"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/validation"
)

func TestAIHandlerStreamWritesPartialChunks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	model := &streamingChatModel{
		chunks: []*einoschema.Message{
			{Role: einoschema.Assistant, Content: "hello "},
			{Role: einoschema.Assistant, Content: "world"},
		},
	}
	service := ai.NewService(config.AIConfig{
		Enabled:      true,
		SystemPrompt: "You are helpful.",
	}, model, nil, nil, nil)
	handler := NewAIHandler(validation.New(), service, nil)

	router := gin.New()
	router.POST("/chat/stream", handler.Stream)

	request := httptest.NewRequest(http.MethodPost, "/chat/stream", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body %s", recorder.Code, body)
	}
	if !strings.Contains(body, `data: {"type":"partial","content":"hello "}`) {
		t.Fatalf("expected first partial chunk, got %s", body)
	}
	if !strings.Contains(body, `data: {"type":"partial","content":"world"}`) {
		t.Fatalf("expected second partial chunk, got %s", body)
	}
	if !strings.Contains(body, `data: {"type":"complete","content":"hello world"}`) {
		t.Fatalf("expected complete chunk, got %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("expected done marker, got %s", body)
	}
}

type streamingChatModel struct {
	chunks []*einoschema.Message
}

func (m *streamingChatModel) Generate(context.Context, []*einoschema.Message) (*einoschema.Message, error) {
	return einoschema.AssistantMessage("fallback", nil), nil
}

func (m *streamingChatModel) Stream(context.Context, []*einoschema.Message) (*einoschema.StreamReader[*einoschema.Message], error) {
	reader, writer := einoschema.Pipe[*einoschema.Message](len(m.chunks))
	go func() {
		defer writer.Close()
		for _, chunk := range m.chunks {
			writer.Send(chunk, nil)
		}
	}()
	return reader, nil
}

func (m *streamingChatModel) BindTools([]*einoschema.ToolInfo) error {
	return nil
}
