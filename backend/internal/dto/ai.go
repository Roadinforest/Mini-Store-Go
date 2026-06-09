package dto

type ChatMessageInput struct {
	Role        string  `json:"role" validate:"required,oneof=user assistant system"`
	Content     string  `json:"content" validate:"required"`
	URL         *string `json:"url,omitempty"`
	MessageType *string `json:"messageType,omitempty"`
	ToolName    *string `json:"toolName,omitempty"`
}

type ChatInput struct {
	Messages []ChatMessageInput `json:"messages" validate:"required,min=1,dive"`
}

type ChatOutput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamChunk struct {
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	URL      string `json:"url,omitempty"`
	Message  string `json:"message,omitempty"`
	ToolName string `json:"toolName,omitempty"`
}
