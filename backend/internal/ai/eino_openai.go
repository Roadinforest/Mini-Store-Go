package ai

import (
	"context"
	"strings"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	einoschema "github.com/cloudwego/eino/schema"

	"mini-store-go/backend/internal/config"
)

type einoChatModel struct {
	model *openai.ChatModel
}

func NewEinoChatModel(ctx context.Context, cfg config.AIConfig) (ChatModel, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil, nil
	}

	var temperature *float32
	temperature = &cfg.Temperature

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		Timeout:     cfg.Timeout,
		Temperature: temperature,
	})
	if err != nil {
		return nil, err
	}

	return &einoChatModel{model: chatModel}, nil
}

func (m *einoChatModel) Generate(ctx context.Context, messages []*einoschema.Message) (*einoschema.Message, error) {
	return m.model.Generate(ctx, messages)
}

func (m *einoChatModel) Stream(ctx context.Context, messages []*einoschema.Message) (*einoschema.StreamReader[*einoschema.Message], error) {
	return m.model.Stream(ctx, messages)
}

func (m *einoChatModel) BindTools(tools []*einoschema.ToolInfo) error {
	return m.model.BindTools(tools)
}
