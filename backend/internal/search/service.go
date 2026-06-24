package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

const rrfK = 60.0

type Service struct {
	cfg      config.SearchConfig
	db       *gorm.DB
	products repository.ProductRepository
	client   *http.Client
}

type Result struct {
	Product model.Product
	Score   float64
}

func NewService(cfg config.SearchConfig, db *gorm.DB, products repository.ProductRepository) *Service {
	return &Service{
		cfg:      cfg,
		db:       db,
		products: products,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (s *Service) SearchProducts(ctx context.Context, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	vectorMatches, _ := s.vectorSearch(ctx, query, 20)
	textMatches, _ := s.textSearch(ctx, query, 20)
	if len(vectorMatches) == 0 && len(textMatches) == 0 {
		return s.repositoryFallback(ctx, query, limit)
	}

	scores := map[string]float64{}
	for i, match := range vectorMatches {
		scores[match.ID] += 1 / (rrfK + float64(i+1))
	}
	for i, match := range textMatches {
		scores[match.ID] += 1 / (rrfK + float64(i+1))
	}

	candidates := rankCandidates(scores, 20)
	products, err := s.productsByID(ctx, candidateIDs(candidates))
	if err != nil {
		return nil, err
	}

	aligned := make([]Result, 0, len(candidates))
	for _, candidate := range candidates {
		product, ok := products[candidate.ID]
		if ok {
			aligned = append(aligned, Result{Product: product, Score: candidate.Score})
		}
	}
	if len(aligned) == 0 {
		return nil, nil
	}

	reranked, err := s.rerank(ctx, query, aligned)
	if err == nil && len(reranked) > 0 {
		aligned = reranked
	}
	sort.SliceStable(aligned, func(i, j int) bool {
		return aligned[i].Score > aligned[j].Score
	})
	if len(aligned) > limit {
		aligned = aligned[:limit]
	}
	return aligned, nil
}

type searchMatch struct {
	ID    string
	Score float64
}

func (s *Service) vectorSearch(ctx context.Context, query string, topK int) ([]searchMatch, error) {
	if !s.cfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(s.cfg.PineconeAPIKey) == "" || strings.TrimSpace(s.cfg.PineconeHost) == "" || strings.TrimSpace(s.cfg.QwenAPIKey) == "" {
		return nil, nil
	}

	embedding, err := s.embedding(ctx, query)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"vector":          embedding,
		"topK":            topK,
		"includeValues":   false,
		"includeMetadata": true,
	}
	var response struct {
		Matches []struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		} `json:"matches"`
	}
	if err := s.postJSON(ctx, strings.TrimRight(s.cfg.PineconeHost, "/")+"/query", map[string]string{
		"Api-Key":      s.cfg.PineconeAPIKey,
		"Content-Type": "application/json",
	}, body, &response); err != nil {
		return nil, err
	}

	matches := make([]searchMatch, 0, len(response.Matches))
	for _, match := range response.Matches {
		matches = append(matches, searchMatch{ID: match.ID, Score: match.Score})
	}
	return matches, nil
}

func (s *Service) embedding(ctx context.Context, text string) ([]float64, error) {
	body := map[string]any{
		"model": s.cfg.EmbeddingModel,
		"input": strings.TrimSpace(text),
	}
	var response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := s.postJSON(ctx, strings.TrimRight(s.cfg.QwenBaseURL, "/")+"/embeddings", map[string]string{
		"Authorization": "Bearer " + s.cfg.QwenAPIKey,
		"Content-Type":  "application/json",
	}, body, &response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 || len(response.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("invalid embedding response")
	}
	return response.Data[0].Embedding, nil
}

func (s *Service) textSearch(ctx context.Context, query string, limit int) ([]searchMatch, error) {
	if s.db == nil {
		return nil, nil
	}
	var rows []struct {
		ID    string  `gorm:"column:id"`
		Score float64 `gorm:"column:score"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, ts_rank(to_tsvector('english', name), websearch_to_tsquery('english', ?)) AS score
		FROM "Product"
		WHERE to_tsvector('english', name) @@ websearch_to_tsquery('english', ?)
		ORDER BY score DESC
		LIMIT ?
	`, query, query, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	matches := make([]searchMatch, 0, len(rows))
	for _, row := range rows {
		matches = append(matches, searchMatch{ID: row.ID, Score: row.Score})
	}
	return matches, nil
}

func (s *Service) rerank(ctx context.Context, query string, results []Result) ([]Result, error) {
	if !s.cfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(s.cfg.QwenAPIKey) == "" {
		return nil, nil
	}

	documents := make([]string, 0, len(results))
	for _, result := range results {
		documents = append(documents, productDocument(result.Product))
	}
	body := map[string]any{
		"model": s.cfg.RerankModel,
		"input": map[string]any{
			"query":     query,
			"documents": documents,
		},
		"parameters": map[string]any{
			"top_n": minInt(len(documents), 10),
		},
	}
	var response struct {
		Output struct {
			Results []struct {
				Index          int     `json:"index"`
				RelevanceScore float64 `json:"relevance_score"`
			} `json:"results"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := s.postJSON(ctx, "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank", map[string]string{
		"Authorization": "Bearer " + s.cfg.QwenAPIKey,
		"Content-Type":  "application/json",
	}, body, &response); err != nil {
		return nil, err
	}
	if response.Code != "" {
		return nil, fmt.Errorf("rerank error %s: %s", response.Code, response.Message)
	}

	reranked := make([]Result, 0, len(response.Output.Results))
	for _, item := range response.Output.Results {
		if item.Index >= 0 && item.Index < len(results) {
			result := results[item.Index]
			result.Score = item.RelevanceScore
			reranked = append(reranked, result)
		}
	}
	return reranked, nil
}

func (s *Service) repositoryFallback(ctx context.Context, query string, limit int) ([]Result, error) {
	if s.products == nil {
		return nil, nil
	}
	items, _, err := s.products.List(ctx, dto.ProductListFilter{
		PageParams: dto.PageParams{Page: 1, Limit: limit},
		Query:      query,
	})
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(items))
	for i, item := range items {
		results = append(results, Result{Product: item, Score: 1 / float64(i+1)})
	}
	return results, nil
}

func (s *Service) productsByID(ctx context.Context, ids []string) (map[string]model.Product, error) {
	if len(ids) == 0 || s.db == nil {
		return map[string]model.Product{}, nil
	}
	var products []model.Product
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]model.Product, len(products))
	for _, product := range products {
		byID[product.ID] = product
	}
	return byID, nil
}

func (s *Service) postJSON(ctx context.Context, url string, headers map[string]string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("post %s failed: status=%d body=%s", url, resp.StatusCode, string(data))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type candidate struct {
	ID    string
	Score float64
}

func rankCandidates(scores map[string]float64, limit int) []candidate {
	candidates := make([]candidate, 0, len(scores))
	for id, score := range scores {
		candidates = append(candidates, candidate{ID: id, Score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func candidateIDs(candidates []candidate) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	return ids
}

func productDocument(product model.Product) string {
	return strings.TrimSpace(strings.Join([]string{
		product.Name,
		product.Brand,
		product.Category,
		product.Description,
	}, " "))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
