package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

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

	stream, err := h.service.Stream(c.Request.Context(), input)
	if err != nil {
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "error",
			Content: "智能助手暂时不可用，请稍后再试。",
		})
		fmt.Fprint(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}
	streamClosed := false
	defer func() {
		if !streamClosed {
			stream.Close()
		}
	}()

	var rawContent strings.Builder
	var visibleContent strings.Builder
	filter := newStreamingThinkFilter()

	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			writeSSEChunk(c, dto.StreamChunk{
				Type:    "error",
				Content: "智能助手暂时不可用，请稍后再试。",
			})
			fmt.Fprint(c.Writer, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}

		if chunk == nil {
			continue
		}

		nextRawContent := rawContent.String() + chunk.Content
		if hasStreamingToolIntent(nextRawContent) || len(chunk.ToolCalls) > 0 {
			stream.Close()
			streamClosed = true
			h.writeNonStreamingChatResult(c, flusher, input)
			return
		}

		rawContent.WriteString(chunk.Content)
		visibleDelta := filter.Push(chunk.Content)
		if visibleDelta == "" {
			continue
		}

		visibleContent.WriteString(visibleDelta)
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "partial",
			Content: visibleDelta,
		})
		flusher.Flush()
	}

	if visibleDelta := filter.Flush(); visibleDelta != "" {
		visibleContent.WriteString(visibleDelta)
		writeSSEChunk(c, dto.StreamChunk{
			Type:    "partial",
			Content: visibleDelta,
		})
		flusher.Flush()
	}

	finalContent := strings.TrimSpace(visibleContent.String())

	h.log.Info("ai stream completion",
		zap.Any("messages", input.Messages),
		zap.String("assistant_raw_content", rawContent.String()),
		zap.String("assistant_visible_content", finalContent),
	)

	writeSSEChunk(c, dto.StreamChunk{
		Type:    "complete",
		Content: finalContent,
	})
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

func (h *AIHandler) writeNonStreamingChatResult(c *gin.Context, flusher interface{ Flush() }, input dto.ChatInput) {
	writeSSEChunk(c, dto.StreamChunk{
		Type:    "tool_call",
		Content: "正在查询商品信息...",
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

	h.log.Info("ai stream fallback completion",
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

func hasStreamingToolIntent(content string) bool {
	return strings.Contains(content, "<tool_call>") || strings.Contains(content, "</tool_call>")
}

type streamingThinkFilter struct {
	inThink bool
	pending string
}

func newStreamingThinkFilter() *streamingThinkFilter {
	return &streamingThinkFilter{}
}

func (f *streamingThinkFilter) Push(content string) string {
	input := f.pending + content
	f.pending = ""

	var output strings.Builder
	for input != "" {
		if f.inThink {
			closeIndex := strings.Index(input, "</think>")
			if closeIndex < 0 {
				f.pending = suffixMatchingPrefix(input, "</think>")
				return output.String()
			}
			input = input[closeIndex+len("</think>"):]
			f.inThink = false
			continue
		}

		openIndex := strings.Index(input, "<")
		if openIndex < 0 {
			output.WriteString(input)
			break
		}
		output.WriteString(input[:openIndex])
		input = input[openIndex:]

		if strings.HasPrefix(input, "<think>") {
			input = input[len("<think>"):]
			f.inThink = true
			continue
		}
		if isHeldMarkupPrefix(input) {
			f.pending = input
			break
		}

		output.WriteByte(input[0])
		input = input[1:]
	}

	return output.String()
}

func (f *streamingThinkFilter) Flush() string {
	if f.inThink {
		f.pending = ""
		return ""
	}
	pending := f.pending
	f.pending = ""
	return pending
}

func suffixMatchingPrefix(value, pattern string) string {
	maxLen := len(pattern) - 1
	if len(value) < maxLen {
		maxLen = len(value)
	}
	for size := maxLen; size > 0; size-- {
		suffix := value[len(value)-size:]
		if strings.HasPrefix(pattern, suffix) {
			return suffix
		}
	}
	return ""
}

func isHeldMarkupPrefix(value string) bool {
	if strings.HasPrefix("<think>", value) {
		return true
	}
	if strings.HasPrefix("<tool_call>", value) {
		return true
	}
	return strings.HasPrefix("</tool_call>", value)
}
