package ai

import (
	"context"
	"strings"
	"testing"

	einoschema "github.com/cloudwego/eino/schema"
	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

func TestParseToolCallsHandlesMinimaxWrappedSearchProducts(t *testing.T) {
	content := `]<]minimax[>[<tool_call>
]<]minimax[>[<invoke name="search_products">]<]minimax[>[<query>苹果手机配件 iPhone accessories]<]minimax[>[</query>
]<]minimax[>[<category>手机配件</category>
]<]minimax[>[<limit>10</limit>
]</invoke>
]</tool_call>`

	calls := parseXMLToolCalls(content)
	if len(calls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(calls))
	}
	if calls[0].Name != toolSearchProducts {
		t.Fatalf("expected %q, got %q", toolSearchProducts, calls[0].Name)
	}
	if calls[0].Params["query"] != "苹果手机配件 iPhone accessories" {
		t.Fatalf("unexpected query param: %q", calls[0].Params["query"])
	}
	if calls[0].Params["category"] != "手机配件" {
		t.Fatalf("unexpected category param: %q", calls[0].Params["category"])
	}
}

func TestChatExecutesSearchProductsToolBeforeFinalAnswer(t *testing.T) {
	chatModel := &fakeChatModel{
		responses: []*einoschema.Message{
			{
				Role:    einoschema.Assistant,
				Content: "<think>private</think>",
				ToolCalls: []einoschema.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: einoschema.FunctionCall{
							Name:      toolSearchProducts,
							Arguments: `{"query":"iPhone case","category":"手机配件","limit":3}`,
						},
					},
				},
			},
			einoschema.AssistantMessage("找到这些真实商品：**iPhone Case**", nil),
		},
	}
	products := &fakeProductRepository{
		items: []model.Product{
			{
				ID:       "p1",
				Name:     "iPhone Case",
				Slug:     "iphone-case",
				Category: "手机配件",
				Price:    decimal.NewFromInt(19),
				Stock:    7,
			},
		},
	}
	service := NewService(config.AIConfig{Enabled: true, MaxContextProducts: 5}, chatModel, products, nil, nil)

	output, err := service.Chat(context.Background(), dto.ChatInput{
		Messages: []dto.ChatMessageInput{{Role: "user", Content: "搜索一下苹果手机相关的配件"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if chatModel.generateCalls != 2 {
		t.Fatalf("expected two model generations, got %d", chatModel.generateCalls)
	}
	if products.lastFilter.Query != "iPhone case" {
		t.Fatalf("expected tool query to be used, got %q", products.lastFilter.Query)
	}
	if strings.Contains(output.Content, "<tool_call>") || strings.Contains(output.Content, "<think>") {
		t.Fatalf("visible output leaked internal markup: %q", output.Content)
	}
	if !strings.Contains(output.Content, "iPhone Case") {
		t.Fatalf("expected final answer, got %q", output.Content)
	}
}

type fakeChatModel struct {
	responses     []*einoschema.Message
	generateCalls int
}

func (m *fakeChatModel) Generate(context.Context, []*einoschema.Message) (*einoschema.Message, error) {
	response := m.responses[m.generateCalls]
	m.generateCalls++
	return response, nil
}

func (m *fakeChatModel) Stream(context.Context, []*einoschema.Message) (*einoschema.StreamReader[*einoschema.Message], error) {
	return nil, nil
}

func (m *fakeChatModel) BindTools([]*einoschema.ToolInfo) error {
	return nil
}

type fakeProductRepository struct {
	items      []model.Product
	lastFilter dto.ProductListFilter
}

func (r *fakeProductRepository) GetByID(context.Context, string) (*model.Product, error) {
	return nil, nil
}

func (r *fakeProductRepository) GetBySlug(context.Context, string) (*model.Product, error) {
	return nil, nil
}

func (r *fakeProductRepository) List(_ context.Context, filter dto.ProductListFilter) ([]model.Product, int64, error) {
	r.lastFilter = filter
	return r.items, int64(len(r.items)), nil
}

func (r *fakeProductRepository) ListLatest(context.Context, int) ([]model.Product, error) {
	return nil, nil
}

func (r *fakeProductRepository) ListFeatured(context.Context, int) ([]model.Product, error) {
	return nil, nil
}

func (r *fakeProductRepository) ListCategories(context.Context) ([]repository.CategoryCount, error) {
	return nil, nil
}

func (r *fakeProductRepository) Create(context.Context, *model.Product) error {
	return nil
}

func (r *fakeProductRepository) Update(context.Context, *model.Product) error {
	return nil
}

func (r *fakeProductRepository) Delete(context.Context, string) error {
	return nil
}
