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
	return &AIHandler{
		validator: validator,
		service:   service,
		log:       log,
	}
}

func (h *AIHandler) Chat(c *gin.Context) {
	var input dto.ChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	output, err := h.service.Chat(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	h.log.Info("ai chat completion",
		zap.Any("messages", input.Messages),
		zap.String("assistant_raw_content", output.RawContent),
		zap.String("assistant_visible_content", output.Content),
	)

	response.OK(c, output)
}

func (h *AIHandler) Stream(c *gin.Context) {
	var input dto.ChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		writeBadRequest(c, "streaming unsupported", nil)
		return
	}

	writeSSEChunk(c, dto.StreamChunk{
		Type:    "thinking",
		Content: "正在思考...",
	})
	flusher.Flush()

	output, err := h.service.Chat(c.Request.Context(), input)
	if err != nil {
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "error",
			Content: "智能助手暂时不可用，请稍后再试。",
		})
		fmt.Fprint(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	h.log.Info("ai stream completion",
		zap.Any("messages", input.Messages),
		zap.String("assistant_raw_content", output.RawContent),
		zap.String("assistant_visible_content", output.Content),
	)

	for _, toolCall := range output.ToolCalls {
		writeSSEChunk(c, dto.StreamChunk{
			Type:     "tool_call",
			Content:  toolCall.Content,
			ToolName: toolCall.ToolName,
		})
		flusher.Flush()
	}

	if output.URL != "" {
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "navigation",
			Content: output.Content,
			URL:     output.URL,
			Message: output.Content,
		})
		fmt.Fprint(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	writeSSEChunk(c, dto.StreamChunk{
		Type:    "partial",
		Content: output.Content,
	})
	writeSSEChunk(c, dto.StreamChunk{
		Type:    "complete",
		Content: output.Content,
	})
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeSSEChunk(c *gin.Context, chunk dto.StreamChunk) {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
}
