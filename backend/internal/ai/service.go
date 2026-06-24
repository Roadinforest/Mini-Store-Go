package ai

import (
	"context"
	"fmt"
	"strings"

	einoschema "github.com/cloudwego/eino/schema"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
	searchsvc "mini-store-go/backend/internal/search"
)

type ChatModel interface {
	Generate(ctx context.Context, messages []*einoschema.Message) (*einoschema.Message, error)
	Stream(ctx context.Context, messages []*einoschema.Message) (*einoschema.StreamReader[*einoschema.Message], error)
	BindTools(tools []*einoschema.ToolInfo) error
}

type Service struct {
	cfg      config.AIConfig
	model    ChatModel
	products repository.ProductRepository
	reviews  repository.ReviewRepository
	search   *searchsvc.Service
}

func NewService(cfg config.AIConfig, model ChatModel, products repository.ProductRepository, reviews repository.ReviewRepository, search *searchsvc.Service) *Service {
	service := &Service{
		cfg:      cfg,
		model:    model,
		products: products,
		reviews:  reviews,
		search:   search,
	}
	if service.model != nil {
		_ = service.model.BindTools(service.toolInfos())
	}
	return service
}

func (s *Service) Chat(ctx context.Context, input dto.ChatInput) (*dto.ChatOutput, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}

	messages, err := s.buildMessages(ctx, input)
	if err != nil {
		return nil, err
	}

	rawParts := []string{}
	toolCalls := []dto.ToolCallOutput{}
	for turn := 0; turn < 8; turn++ {
		msg, err := s.model.Generate(ctx, messages)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to generate chat response", err)
		}
		rawParts = append(rawParts, msg.Content)

		executions, err := s.executeToolCalls(ctx, msg)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to execute ai tool call", err)
		}
		if len(executions) == 0 {
			return &dto.ChatOutput{
				Role:       string(einoschema.Assistant),
				Content:    VisibleContent(msg.Content),
				ToolCalls:  toolCalls,
				RawContent: strings.Join(rawParts, "\n\n"),
			}, nil
		}
		toolCalls = append(toolCalls, toToolCallOutputs(executions)...)

		if navigation := firstNavigation(executions); navigation != nil {
			return &dto.ChatOutput{
				Role:        string(einoschema.Assistant),
				Content:     navigation.Message,
				RawContent:  strings.Join(rawParts, "\n\n"),
				URL:         navigation.URL,
				MessageType: ptrString("navigation"),
				ToolCalls:   toolCalls,
			}, nil
		}

		messages = appendToolResultMessages(messages, msg, executions)
	}

	return nil, apperror.New(apperror.CodeInternal, "ai agent exceeded maximum tool turns")
}

func (s *Service) Stream(ctx context.Context, input dto.ChatInput) (*einoschema.StreamReader[*einoschema.Message], error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}

	messages, err := s.buildMessages(ctx, input)
	if err != nil {
		return nil, err
	}

	stream, err := s.model.Stream(ctx, messages)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to stream chat response", err)
	}

	return stream, nil
}

func (s *Service) ensureAvailable() error {
	if !s.cfg.Enabled || s.model == nil {
		return apperror.New(apperror.CodeServiceDisabled, "ai service is disabled")
	}
	return nil
}

func (s *Service) buildMessages(ctx context.Context, input dto.ChatInput) ([]*einoschema.Message, error) {
	messages := make([]*einoschema.Message, 0, len(input.Messages)+2)
	messages = append(messages, einoschema.SystemMessage(s.cfg.SystemPrompt))

	if prompt := s.buildContextPrompt(ctx, input); prompt != "" {
		messages = append(messages, einoschema.SystemMessage(prompt))
	}

	for _, msg := range input.Messages {
		messages = append(messages, toEinoMessage(msg))
	}

	return messages, nil
}

func (s *Service) buildContextPrompt(ctx context.Context, input dto.ChatInput) string {
	if s.products == nil || s.cfg.MaxContextProducts <= 0 {
		return ""
	}

	query := latestUserMessage(input.Messages)
	if query == "" {
		return ""
	}

	items, _, err := s.products.List(ctx, dto.ProductListFilter{
		PageParams: dto.PageParams{Page: 1, Limit: s.cfg.MaxContextProducts},
		Query:      query,
	})
	if err != nil || len(items) == 0 {
		return ""
	}

	lines := []string{
		"Relevant catalog context:",
	}
	for _, product := range items {
		lines = append(lines, formatProductContext(product))
	}

	return strings.Join(lines, "\n")
}

func latestUserMessage(messages []dto.ChatMessageInput) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && strings.TrimSpace(messages[i].Content) != "" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func toEinoMessage(msg dto.ChatMessageInput) *einoschema.Message {
	switch msg.Role {
	case string(einoschema.System):
		return einoschema.SystemMessage(msg.Content)
	case string(einoschema.Assistant):
		return einoschema.AssistantMessage(msg.Content, nil)
	default:
		return einoschema.UserMessage(msg.Content)
	}
}

func formatProductContext(product model.Product) string {
	parts := []string{
		fmt.Sprintf("- %s", product.Name),
		fmt.Sprintf("slug=%s", product.Slug),
		fmt.Sprintf("category=%s", product.Category),
		fmt.Sprintf("price=%s", formatDecimal(product.Price)),
		fmt.Sprintf("stock=%d", product.Stock),
	}

	if !product.Rating.IsZero() {
		parts = append(parts, fmt.Sprintf("rating=%s", formatDecimal(product.Rating)))
	}
	if product.Description != "" {
		parts = append(parts, fmt.Sprintf("description=%s", compactText(product.Description, 220)))
	}

	return strings.Join(parts, " | ")
}

func compactText(value string, maxLen int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen-3] + "..."
}

func firstNavigation(executions []toolExecution) *navigationResult {
	for _, execution := range executions {
		if execution.Result.Navigation != nil {
			return execution.Result.Navigation
		}
	}
	return nil
}

func ptrString(value string) *string {
	return &value
}

func toToolCallOutputs(executions []toolExecution) []dto.ToolCallOutput {
	outputs := make([]dto.ToolCallOutput, 0, len(executions))
	for _, execution := range executions {
		outputs = append(outputs, dto.ToolCallOutput{
			ToolName: execution.Call.Name,
			Content:  execution.Hint,
		})
	}
	return outputs
}
