package ai

import "strings"

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
)

func VisibleContent(content string) string {
	content = stripToolMarkup(content)

	var builder strings.Builder
	remaining := content

	for {
		openIndex := strings.Index(remaining, thinkOpenTag)
		if openIndex < 0 {
			builder.WriteString(remaining)
			break
		}

		builder.WriteString(remaining[:openIndex])
		afterOpen := remaining[openIndex+len(thinkOpenTag):]
		closeIndex := strings.Index(afterOpen, thinkCloseTag)
		if closeIndex < 0 {
			break
		}

		remaining = afterOpen[closeIndex+len(thinkCloseTag):]
	}

	return strings.TrimSpace(builder.String())
}
