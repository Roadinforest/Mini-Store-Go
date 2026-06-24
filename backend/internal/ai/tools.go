package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	einoschema "github.com/cloudwego/eino/schema"
	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
)

const (
	toolSearchProducts       = "search_products"
	toolSearchProductsByName = "search_products_by_name"
	toolHybridSearchProducts = "hybrid_search_products"
	toolGetProductDetails    = "get_product_details"
	toolGetProductReviews    = "get_product_reviews"
	toolGetAllProductNames   = "get_all_product_names"
	toolSearchAgent          = "search_agent"
	toolReviewAgent          = "review_agent"
)

var (
	invokePattern    = regexp.MustCompile(`(?s)<invoke\s+name=["']([^"']+)["']>(.*?)</invoke>`)
	toolCallPattern  = regexp.MustCompile(`(?s)<tool_call>.*?</tool_call>`)
	toolParamPattern = regexp.MustCompile(`(?s)<([a-zA-Z_][a-zA-Z0-9_]*)>(.*?)</([a-zA-Z_][a-zA-Z0-9_]*)>`)
)

type toolDefinition struct {
	info *einoschema.ToolInfo
	hint func(map[string]any) string
	run  func(context.Context, map[string]any) (toolResult, error)
}

type toolCall struct {
	ID     string
	Name   string
	Params map[string]any
}

type toolExecution struct {
	Call   toolCall
	Result toolResult
	Hint   string
}

type toolResult struct {
	Content    string
	Navigation *navigationResult
}

type navigationResult struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}

func (s *Service) toolDefinitions() []toolDefinition {
	return []toolDefinition{
		s.searchProductsTool(toolSearchProducts, "搜索商品，支持关键词、分类、排序和数量限制"),
		s.searchProductsTool(toolSearchProductsByName, "根据产品名称搜索相关产品"),
		s.hybridSearchProductsTool(),
		s.getProductDetailsTool(),
		s.getProductReviewsTool(),
		s.getAllProductNamesTool(),
		s.searchAgentTool(),
		s.reviewAgentTool(),
	}
}

func (s *Service) hybridSearchProductsTool() toolDefinition {
	definition := s.searchProductsTool(toolHybridSearchProducts, "使用混合搜索技术（向量搜索 + 全文检索 + RRF融合 + 重排序）搜索产品信息")
	definition.run = func(ctx context.Context, args map[string]any) (toolResult, error) {
		if s.search == nil {
			return s.executeSearchProducts(ctx, args)
		}
		query := stringArg(args, "query")
		limit := parseToolLimit(args["limit"], 10)
		results, err := s.search.SearchProducts(ctx, query, limit)
		if err != nil || len(results) == 0 {
			fallback, fallbackErr := s.executeSearchProducts(ctx, args)
			if fallbackErr != nil {
				return toolResult{}, fallbackErr
			}
			return fallback, nil
		}

		lines := []string{fmt.Sprintf("混合搜索结果 (共找到%d个产品，展示%d个):", len(results), len(results))}
		for i, result := range results {
			lines = append(lines, fmt.Sprintf("产品 %d:\n%s | score=%.4f", i+1, formatProductContext(result.Product), result.Score))
		}
		return textToolResult(strings.Join(lines, "\n\n")), nil
	}
	return definition
}

func (s *Service) toolInfos() []*einoschema.ToolInfo {
	definitions := s.toolDefinitions()
	infos := make([]*einoschema.ToolInfo, 0, len(definitions))
	for _, definition := range definitions {
		infos = append(infos, definition.info)
	}
	return infos
}

func (s *Service) executeToolCalls(ctx context.Context, msg *einoschema.Message) ([]toolExecution, error) {
	calls := parseMessageToolCalls(msg)
	if len(calls) == 0 {
		return nil, nil
	}

	definitions := map[string]toolDefinition{}
	for _, definition := range s.toolDefinitions() {
		definitions[definition.info.Name] = definition
	}

	executions := make([]toolExecution, 0, len(calls))
	for _, call := range calls {
		definition, ok := definitions[call.Name]
		if !ok {
			continue
		}
		result, err := definition.run(ctx, call.Params)
		if err != nil {
			return nil, err
		}
		executions = append(executions, toolExecution{
			Call:   call,
			Result: result,
			Hint:   toolHint(definition, call),
		})
	}

	return executions, nil
}

func toolHint(definition toolDefinition, call toolCall) string {
	if definition.hint != nil {
		if hint := strings.TrimSpace(definition.hint(call.Params)); hint != "" {
			return hint
		}
	}
	return fmt.Sprintf("正在调用工具: %s", call.Name)
}

func parseMessageToolCalls(msg *einoschema.Message) []toolCall {
	if msg == nil {
		return nil
	}

	calls := make([]toolCall, 0, len(msg.ToolCalls))
	for _, call := range msg.ToolCalls {
		params := map[string]any{}
		if strings.TrimSpace(call.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(call.Function.Arguments), &params)
		}
		calls = append(calls, toolCall{
			ID:     call.ID,
			Name:   strings.TrimSpace(call.Function.Name),
			Params: params,
		})
	}
	if len(calls) > 0 {
		return calls
	}

	return parseXMLToolCalls(msg.Content)
}

func parseXMLToolCalls(content string) []toolCall {
	cleaned := normalizeToolMarkup(content)
	matches := invokePattern.FindAllStringSubmatch(cleaned, -1)
	calls := make([]toolCall, 0, len(matches))

	for _, match := range matches {
		call := toolCall{
			Name:   strings.TrimSpace(match[1]),
			Params: map[string]any{},
		}
		for _, paramMatch := range toolParamPattern.FindAllStringSubmatch(match[2], -1) {
			if paramMatch[1] != paramMatch[3] {
				continue
			}
			call.Params[strings.TrimSpace(paramMatch[1])] = strings.TrimSpace(paramMatch[2])
		}
		calls = append(calls, call)
	}

	return calls
}

func (s *Service) searchProductsTool(name, description string) toolDefinition {
	return toolDefinition{
		info: &einoschema.ToolInfo{
			Name: name,
			Desc: description,
			ParamsOneOf: einoschema.NewParamsOneOfByParams(map[string]*einoschema.ParameterInfo{
				"query":    {Type: einoschema.String, Desc: "搜索关键词", Required: true},
				"category": {Type: einoschema.String, Desc: "商品分类"},
				"sort_by":  {Type: einoschema.String, Desc: "排序方式", Enum: []string{"lowest", "highest", "rating", "popularity"}},
				"limit":    {Type: einoschema.Integer, Desc: "返回结果数量，最大 20"},
				"method":   {Type: einoschema.String, Desc: "搜索方法", Enum: []string{"name", "rag", "auto"}},
			}),
		},
		hint: func(args map[string]any) string {
			return fmt.Sprintf("正在搜索与 %q 相关的商品", stringArg(args, "query"))
		},
		run: func(ctx context.Context, args map[string]any) (toolResult, error) {
			return s.executeSearchProducts(ctx, args)
		},
	}
}

func (s *Service) getProductDetailsTool() toolDefinition {
	return toolDefinition{
		info: &einoschema.ToolInfo{
			Name: toolGetProductDetails,
			Desc: "根据产品ID获取详细的产品信息",
			ParamsOneOf: einoschema.NewParamsOneOfByParams(map[string]*einoschema.ParameterInfo{
				"productId": {Type: einoschema.String, Desc: "产品ID", Required: true},
			}),
		},
		run: func(ctx context.Context, args map[string]any) (toolResult, error) {
			if s.products == nil {
				return textToolResult("商品详情工具不可用"), nil
			}
			productID := stringArg(args, "productId")
			product, err := s.products.GetByID(ctx, productID)
			if err != nil || product == nil {
				return textToolResult(fmt.Sprintf("未找到ID为 %s 的产品", productID)), nil
			}
			return textToolResult(formatProductDetails(*product)), nil
		},
	}
}

func (s *Service) getProductReviewsTool() toolDefinition {
	return toolDefinition{
		info: &einoschema.ToolInfo{
			Name: toolGetProductReviews,
			Desc: "根据产品ID获取该产品的所有用户评论",
			ParamsOneOf: einoschema.NewParamsOneOfByParams(map[string]*einoschema.ParameterInfo{
				"productId": {Type: einoschema.String, Desc: "产品ID", Required: true},
			}),
		},
		run: func(ctx context.Context, args map[string]any) (toolResult, error) {
			return s.executeGetProductReviews(ctx, stringArg(args, "productId"))
		},
	}
}

func (s *Service) getAllProductNamesTool() toolDefinition {
	return toolDefinition{
		info: &einoschema.ToolInfo{
			Name: toolGetAllProductNames,
			Desc: "获取商店中所有产品的名称、ID、分类和品牌信息",
			ParamsOneOf: einoschema.NewParamsOneOfByParams(map[string]*einoschema.ParameterInfo{
				"limit": {Type: einoschema.Integer, Desc: "返回结果数量，默认100"},
			}),
		},
		run: func(ctx context.Context, args map[string]any) (toolResult, error) {
			limit := parseToolLimit(args["limit"], 100)
			if limit > 100 {
				limit = 100
			}
			result, err := s.executeSearchProducts(ctx, map[string]any{"query": "all", "limit": limit})
			return result, err
		},
	}
}

func (s *Service) searchAgentTool() toolDefinition {
	definition := s.searchProductsTool(toolSearchAgent, "调用专门的搜索代理来执行复杂的产品搜索任务")
	definition.hint = func(map[string]any) string { return "Search Agent 全力工作中……" }
	return definition
}

func (s *Service) reviewAgentTool() toolDefinition {
	return toolDefinition{
		info: &einoschema.ToolInfo{
			Name: toolReviewAgent,
			Desc: "调用专门的评论分析代理来获取和分析产品评论",
			ParamsOneOf: einoschema.NewParamsOneOfByParams(map[string]*einoschema.ParameterInfo{
				"productId": {Type: einoschema.String, Desc: "产品ID", Required: true},
				"analysis":  {Type: einoschema.Boolean, Desc: "是否需要分析评论内容"},
				"summary":   {Type: einoschema.Boolean, Desc: "是否需要评论摘要"},
			}),
		},
		run: func(ctx context.Context, args map[string]any) (toolResult, error) {
			return s.executeGetProductReviews(ctx, stringArg(args, "productId"))
		},
	}
}

func (s *Service) executeSearchProducts(ctx context.Context, args map[string]any) (toolResult, error) {
	if s.products == nil {
		return textToolResult("search_products is unavailable because the product repository is not configured."), nil
	}

	limit := parseToolLimit(args["limit"], s.cfg.MaxContextProducts)
	filter := dto.ProductListFilter{
		PageParams: dto.PageParams{Page: 1, Limit: limit},
		Query:      stringArg(args, "query"),
		Category:   stringArg(args, "category"),
		Sort:       normalizeProductSort(stringArg(args, "sort_by")),
	}

	items, total, err := s.products.List(ctx, filter)
	if err != nil {
		return toolResult{}, err
	}
	if len(items) == 0 && strings.TrimSpace(filter.Category) != "" {
		filter.Category = ""
		items, total, err = s.products.List(ctx, filter)
		if err != nil {
			return toolResult{}, err
		}
	}

	return textToolResult(formatSearchProductsResult(filter, items, total)), nil
}

func (s *Service) executeGetProductReviews(ctx context.Context, productID string) (toolResult, error) {
	if s.reviews == nil {
		return textToolResult("评论工具不可用"), nil
	}
	reviews, err := s.reviews.ListByProductID(ctx, productID)
	if err != nil {
		return toolResult{}, err
	}
	if len(reviews) == 0 {
		return textToolResult(fmt.Sprintf("产品ID %s 暂无评论", productID)), nil
	}

	lines := []string{fmt.Sprintf("产品评论 (共%d条):", len(reviews))}
	for i, review := range reviews {
		userName := review.User.Name
		if strings.TrimSpace(userName) == "" {
			userName = "匿名用户"
		}
		lines = append(lines, fmt.Sprintf("评论 %d:\n用户: %s\n评分: %d/5 星\n标题: %s\n内容: %s\n创建时间: %s",
			i+1, userName, review.Rating, review.Title, review.Description, review.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	return textToolResult(strings.Join(lines, "\n\n")), nil
}

func parseToolLimit(value any, fallback int) int {
	if fallback <= 0 {
		fallback = 5
	}

	var limit int
	switch typed := value.(type) {
	case int:
		limit = typed
	case int64:
		limit = int(typed)
	case float64:
		limit = int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		limit = int(parsed)
	case string:
		_, _ = fmt.Sscanf(typed, "%d", &limit)
	}
	if limit <= 0 {
		return fallback
	}
	if limit > 20 {
		return 20
	}
	return limit
}

func normalizeProductSort(value string) string {
	switch strings.TrimSpace(value) {
	case "lowest", "highest", "rating":
		return value
	default:
		return ""
	}
}

func formatSearchProductsResult(filter dto.ProductListFilter, items []model.Product, total int64) string {
	if len(items) == 0 {
		return fmt.Sprintf("没有找到与 query=%q category=%q 相关的产品。", filter.Query, filter.Category)
	}

	lines := []string{
		fmt.Sprintf("搜索结果 (共找到%d个产品，展示%d个):", total, len(items)),
	}
	for i, item := range items {
		lines = append(lines, fmt.Sprintf("产品 %d:\n%s", i+1, formatProductContext(item)))
	}

	return strings.Join(lines, "\n\n")
}

func formatProductDetails(product model.Product) string {
	return fmt.Sprintf("产品详情：\n名称: %s\nID: %s\n描述: %s\n价格: $%s\n分类: %s\n库存: %d\n评分: %s\n评论数: %d\n是否特色产品: %t\n品牌: %s\n图片: %s\n创建时间: %s",
		product.Name,
		product.ID,
		product.Description,
		formatDecimal(product.Price),
		product.Category,
		product.Stock,
		formatDecimal(product.Rating),
		product.NumReviews,
		product.IsFeatured,
		product.Brand,
		strings.Join(product.Images, ", "),
		product.CreatedAt.Format("2006-01-02 15:04:05"),
	)
}

func appendToolResultMessages(messages []*einoschema.Message, firstResponse *einoschema.Message, executions []toolExecution) []*einoschema.Message {
	next := make([]*einoschema.Message, 0, len(messages)+1+len(executions)+1)
	next = append(next, messages...)
	next = append(next, firstResponse)

	for _, execution := range executions {
		if execution.Call.ID != "" {
			next = append(next, einoschema.ToolMessage(execution.Result.Content, execution.Call.ID, einoschema.WithToolName(execution.Call.Name)))
			continue
		}
		next = append(next, einoschema.SystemMessage(fmt.Sprintf("Tool: %s\n%s", execution.Call.Name, execution.Result.Content)))
	}
	next = append(next, einoschema.SystemMessage("Use only these Mini Store backend tool results to answer the user. Do not expose tool XML, function call JSON, internal tool markup, or invented products."))
	return next
}

func stripToolMarkup(content string) string {
	return strings.TrimSpace(toolCallPattern.ReplaceAllString(normalizeToolMarkup(content), ""))
}

func normalizeToolMarkup(content string) string {
	return strings.ReplaceAll(content, "]<]minimax[>[", "")
}

func textToolResult(content string) toolResult {
	return toolResult{Content: content}
}

func stringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func boolArg(args map[string]any, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func formatDecimal(value decimal.Decimal) string {
	return value.StringFixedBank(2)
}
