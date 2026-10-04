package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"mini-store-go/backend/internal/ai"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/response"
	"mini-store-go/backend/internal/validation"
	"time"
)

type AIHandler struct {
	validator *validation.Validator
	service   *ai.Service
	log       *zap.Logger
}

func NewAIHandler(validator *validation.Validator, service *ai.Service, log *zap.Logger) *AIHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &AIHandler{validator, service, log}
}
func (h *AIHandler) input(c *gin.Context) (dto.ChatInput, bool) {
	var input dto.ChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return input, false
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return input, false
	}
	return input, true
}
func (h *AIHandler) Chat(c *gin.Context) {
	input, ok := h.input(c)
	if !ok {
		return
	}
	started := time.Now()
	output, err := h.service.Chat(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	h.completion(c, output, started, false)
	response.OK(c, output)
}
func (h *AIHandler) Stream(c *gin.Context) {
	input, ok := h.input(c)
	if !ok {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	emit := func(chunk dto.StreamChunk) error {
		if err := c.Request.Context().Err(); err != nil {
			return err
		}
		payload, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		if _, err = fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	started := time.Now()
	output, err := h.service.Stream(c.Request.Context(), input, emit)
	if err != nil {
		h.log.Warn("ai stream failed", zap.String("request_id", c.GetString("request_id")), zap.Duration("duration", time.Since(started)))
		if c.Request.Context().Err() != nil {
			return
		}
		_ = emit(dto.StreamChunk{Type: "error", Content: "智能助手暂时不可用，请稍后再试。"})
	} else {
		h.completion(c, output, started, true)
		chunk := dto.StreamChunk{Type: "complete", Content: output.Content}
		if output.URL != "" {
			chunk.Type = "navigation"
			chunk.URL = output.URL
			chunk.Message = output.Content
		}
		if err := emit(chunk); err != nil {
			return
		}
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}
func (h *AIHandler) completion(c *gin.Context, output *dto.ChatOutput, started time.Time, streamed bool) {
	h.log.Info("ai completion", zap.String("request_id", c.GetString("request_id")), zap.Bool("streamed", streamed), zap.Duration("duration", time.Since(started)), zap.Int("tool_calls", len(output.ToolCalls)), zap.Int("response_bytes", len(output.Content)))
}
