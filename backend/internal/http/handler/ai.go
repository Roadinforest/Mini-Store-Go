package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	einoschema "github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/ai"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/response"
	"mini-store-go/backend/internal/validation"
)

type AIHandler struct {
	validator *validation.Validator
	service   *ai.Service
}

func NewAIHandler(validator *validation.Validator, service *ai.Service) *AIHandler {
	return &AIHandler{
		validator: validator,
		service:   service,
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

	stream, err := h.service.Stream(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	defer stream.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		writeBadRequest(c, "streaming unsupported", nil)
		return
	}

	var contentBuilder strings.Builder
	for {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			writeSSEChunk(c, dto.StreamChunk{
				Type:    "error",
				Content: "抱歉，我遇到了一些问题。请稍后再试。",
			})
			flusher.Flush()
			return
		}

		writeMessageChunks(c, msg, &contentBuilder)
		flusher.Flush()
	}

	writeSSEChunk(c, dto.StreamChunk{
		Type:    "complete",
		Content: contentBuilder.String(),
	})
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeMessageChunks(c *gin.Context, msg *einoschema.Message, contentBuilder *strings.Builder) {
	if msg == nil {
		return
	}

	if strings.TrimSpace(msg.ReasoningContent) != "" {
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "thinking",
			Content: msg.ReasoningContent,
		})
	}

	if strings.TrimSpace(msg.Content) != "" {
		contentBuilder.WriteString(msg.Content)
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "partial",
			Content: msg.Content,
		})
	}
}

func writeSSEChunk(c *gin.Context, chunk dto.StreamChunk) {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
}
