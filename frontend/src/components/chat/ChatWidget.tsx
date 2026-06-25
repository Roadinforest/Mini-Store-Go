import { LoaderCircle, MessageCircle, SendHorizontal, X } from "lucide-react";
import { FormEvent, ReactNode, useEffect, useRef, useState } from "react";
import { createChatStream, sendChat, type ChatMessage, type ChatStreamChunk } from "@/lib/api";

const INITIAL_MESSAGES: ChatMessage[] = [
  {
    role: "assistant",
    content: "你好，我是 Mini Store 智能助手。你可以问我商品推荐、下单流程或配送相关问题。",
  },
];

export function ChatWidget() {
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>(INITIAL_MESSAGES);
  const [error, setError] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages, isLoading]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const content = input.trim();
    if (!content || isLoading) return;

    const userMessage: ChatMessage = {
      role: "user",
      content,
    };

    const nextMessages = [...messages, userMessage];
    setMessages([
      ...nextMessages,
      {
        role: "assistant",
        content: "正在思考...",
        messageType: "thinking",
      },
    ]);
    setInput("");
    setError(null);
    setIsLoading(true);

    try {
      const streamed = await streamChat(nextMessages);
      if (!streamed) {
        await sendFallbackChat(nextMessages);
      }
    } catch {
      const message = "无法连接到后端服务，请检查 API 地址、CORS 配置或后端是否正在运行。";
      setError(message);
      setMessages((current) => replaceLastAssistant(current, { content: "智能助手暂时不可用，请稍后再试。", messageType: "normal" }));
    } finally {
      setIsLoading(false);
    }
  }

  async function sendFallbackChat(nextMessages: ChatMessage[]) {
    const fallback = await sendChat(nextMessages);
    if (!fallback.success || !fallback.data) {
      const message = friendlyChatError(fallback.message);
      setError(message);
      setMessages((current) =>
        replaceLastAssistant(current, {
          content: "智能助手暂时不可用，请稍后再试。",
          messageType: "normal",
        }),
      );
      return;
    }

    const toolHint = latestToolHint(fallback.data);
    if (toolHint) {
      setMessages((current) =>
        replaceLastAssistant(current, {
          content: toolHint,
          messageType: "tool_call",
          toolName: fallback.data?.toolCalls?.at(-1)?.toolName,
        }),
      );
      await sleep(450);
    }
    const visibleFallback = {
      ...fallback.data,
      content: visibleAssistantContent(fallback.data.content),
      messageType: fallback.data.messageType ?? "normal",
    };
    setMessages((current) => replaceLastAssistant(current, visibleFallback));
  }

  async function streamChat(nextMessages: ChatMessage[]) {
    const response = await createChatStream(nextMessages);
    if (!response.ok || !response.body) {
      return false;
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let assistantContent = "";

    async function handleChunk(chunk: ChatStreamChunk) {
      if (chunk.type === "thinking") {
        setMessages((current) =>
          replaceLastAssistant(current, {
            content: chunk.content ?? "正在思考...",
            messageType: "thinking",
          }),
        );
        return;
      }

      if (chunk.type === "tool_call") {
        assistantContent = "";
        setMessages((current) =>
          replaceLastAssistant(current, {
            content: chunk.content ?? "正在查询商品信息...",
            messageType: "tool_call",
            toolName: chunk.toolName,
          }),
        );
        return;
      }

      if (chunk.type === "partial") {
        assistantContent += chunk.content ?? "";
        setMessages((current) =>
          replaceLastAssistant(current, {
            content: visibleAssistantContent(assistantContent),
            messageType: "normal",
          }),
        );
        return;
      }

      if (chunk.type === "complete") {
        assistantContent = chunk.content ?? assistantContent;
        setMessages((current) =>
          replaceLastAssistant(current, {
            content: visibleAssistantContent(assistantContent),
            messageType: "normal",
          }),
        );
        return;
      }

      if (chunk.type === "navigation") {
        setMessages((current) =>
          replaceLastAssistant(current, {
            content: chunk.message ?? chunk.content ?? "",
            messageType: "navigation",
            url: chunk.url,
          }),
        );
        return;
      }

      if (chunk.type === "error") {
        throw new Error(chunk.content ?? "stream error");
      }
    }

    async function drainEvents() {
      const events = buffer.split("\n\n");
      buffer = events.pop() ?? "";

      for (const event of events) {
        const dataLines = event
          .split("\n")
          .filter((line) => line.startsWith("data:"))
          .map((line) => line.slice(5).trimStart());
        if (dataLines.length === 0) continue;

        const data = dataLines.join("\n").trim();
        if (!data || data === "[DONE]") {
          continue;
        }

        await handleChunk(JSON.parse(data) as ChatStreamChunk);
      }
    }

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      await drainEvents();
    }

    buffer += decoder.decode();
    await drainEvents();
    return true;
  }

  return (
    <>
      {isOpen && (
        <div className="fixed inset-x-4 bottom-24 z-40 mx-auto flex h-[34rem] max-w-md flex-col overflow-hidden rounded-3xl border bg-background shadow-2xl md:right-6 md:left-auto md:mx-0">
          <div className="flex items-center justify-between border-b px-5 py-4">
            <div>
              <div className="text-sm font-semibold">智能助手</div>
              <div className="text-xs text-muted-foreground">支持商品推荐与购物问题咨询</div>
            </div>
            <button
              type="button"
              className="rounded-full p-2 text-muted-foreground transition hover:bg-muted hover:text-foreground"
              onClick={() => setIsOpen(false)}
              aria-label="Close chat"
            >
              <X className="size-4" />
            </button>
          </div>

          <div ref={scrollRef} className="flex-1 space-y-4 overflow-y-auto bg-muted/30 px-4 py-4">
            {messages.map((message, index) => (
              <div
                key={`${message.role}-${index}`}
                className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-6 shadow-sm ${
                    message.role === "user"
                      ? "rounded-br-sm bg-primary text-primary-foreground"
                      : "rounded-bl-sm border bg-background text-foreground"
                  }`}
                >
                  <ChatMessageBody
                    content={message.content || (isLoading && index === messages.length - 1 ? "正在思考..." : "")}
                    isPending={isLoading && index === messages.length - 1 && message.role === "assistant" && !message.content}
                    messageType={message.messageType}
                  />
                </div>
              </div>
            ))}
            {error && <div className="text-center text-xs text-red-500">{error}</div>}
          </div>

          <form onSubmit={onSubmit} className="border-t bg-background p-4">
            <div className="flex items-end gap-3">
              <textarea
                rows={2}
                className="min-h-20 flex-1 resize-none rounded-2xl border bg-background px-4 py-3 text-sm outline-none transition focus:border-primary"
                placeholder="输入你的问题..."
                value={input}
                onChange={(event) => setInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && !event.shiftKey) {
                    event.preventDefault();
                    event.currentTarget.form?.requestSubmit();
                  }
                }}
              />
              <button
                type="submit"
                className="inline-flex h-11 w-11 items-center justify-center rounded-full bg-primary text-primary-foreground transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
                disabled={isLoading || !input.trim()}
                aria-label="Send message"
              >
                <SendHorizontal className="size-4" />
              </button>
            </div>
          </form>
        </div>
      )}

      <button
        type="button"
        onClick={() => setIsOpen((current) => !current)}
        className="fixed bottom-6 right-6 z-40 inline-flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-xl transition hover:scale-[1.02] hover:bg-primary/90"
        aria-label="Open chat assistant"
      >
        <MessageCircle className="size-6" />
      </button>
    </>
  );
}

function replaceLastAssistant(messages: ChatMessage[], message: Pick<ChatMessage, "content"> & Partial<ChatMessage>) {
  const next = [...messages];
  const lastIndex = next.length - 1;
  if (lastIndex < 0 || next[lastIndex]?.role !== "assistant") {
    next.push({ role: "assistant", ...message });
    return next;
  }

  next[lastIndex] = {
    ...next[lastIndex],
    ...message,
  };
  return next;
}

function visibleAssistantContent(content: string) {
  return content.replace(/<think>[\s\S]*?(?:<\/think>|$)/g, "").trim();
}

function latestToolHint(message: ChatMessage) {
  return message.toolCalls?.at(-1)?.content;
}

function sleep(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function friendlyChatError(message?: string) {
  if (!message || message === "Failed to fetch" || message === "NetworkError when attempting to fetch resource.") {
    return "无法连接到后端服务，请检查 API 地址、CORS 配置或后端是否正在运行。";
  }
  return message;
}

function ChatMessageBody({
  content,
  isPending = false,
  messageType,
}: {
  content: string;
  isPending?: boolean;
  messageType?: ChatMessage["messageType"];
}) {
  if (isPending || messageType === "thinking" || messageType === "tool_call") {
    return <StatusMessage content={content} />;
  }

  const blocks = parseMarkdownBlocks(content);

  if (blocks.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      {blocks.map((block, index) => {
        if (block.type === "heading") {
          const HeadingTag = `h${block.level}` as const;
          return (
            <HeadingTag key={index} className="pt-1 text-sm font-semibold leading-6">
              {renderInlineMarkdown(block.text)}
            </HeadingTag>
          );
        }

        if (block.type === "list") {
          return (
            <ul key={index} className="list-disc space-y-1 pl-5">
              {block.items.map((item, itemIndex) => (
                <li key={itemIndex}>{renderInlineMarkdown(item)}</li>
              ))}
            </ul>
          );
        }

        if (block.type === "quote") {
          return (
            <blockquote key={index} className="border-l-2 border-border pl-3 text-muted-foreground">
              {renderInlineMarkdown(block.text)}
            </blockquote>
          );
        }

        if (block.type === "table") {
          return (
            <div key={index} className="overflow-x-auto">
              <table className="min-w-full border-collapse text-left text-xs">
                <thead>
                  <tr>
                    {block.headers.map((header, headerIndex) => (
                      <th key={headerIndex} className="border-b px-2 py-1 font-semibold">
                        {renderInlineMarkdown(header)}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {block.rows.map((row, rowIndex) => (
                    <tr key={rowIndex}>
                      {row.map((cell, cellIndex) => (
                        <td key={cellIndex} className="border-b px-2 py-1 align-top">
                          {renderInlineMarkdown(cell)}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        }

        if (block.type === "divider") {
          return <hr key={index} className="border-border" />;
        }

        return (
          <p key={index} className="whitespace-pre-wrap">
            {renderInlineMarkdown(block.text)}
          </p>
        );
      })}
    </div>
  );
}

function StatusMessage({ content }: { content: string }) {
  return (
    <div className="inline-flex items-center gap-2 text-muted-foreground">
      <LoaderCircle className="size-4 animate-spin" />
      <span>{content}</span>
    </div>
  );
}

type MarkdownBlock =
  | {
      type: "paragraph";
      text: string;
    }
  | {
      type: "heading";
      level: 1 | 2 | 3;
      text: string;
    }
  | {
      type: "list";
      items: string[];
    }
  | {
      type: "quote";
      text: string;
    }
  | {
      type: "table";
      headers: string[];
      rows: string[][];
    }
  | {
      type: "divider";
    };

function parseMarkdownBlocks(content: string): MarkdownBlock[] {
  const blocks: MarkdownBlock[] = [];
  const lines = content.replace(/\r\n/g, "\n").split("\n");
  let paragraph: string[] = [];
  let list: string[] = [];
  let index = 0;

  function flushParagraph() {
    if (paragraph.length === 0) return;
    blocks.push({ type: "paragraph", text: paragraph.join("\n").trim() });
    paragraph = [];
  }

  function flushList() {
    if (list.length === 0) return;
    blocks.push({ type: "list", items: list });
    list = [];
  }

  while (index < lines.length) {
    const line = lines[index];

    if (isMarkdownTableStart(lines, index)) {
      flushParagraph();
      flushList();
      const headers = splitTableRow(line);
      index += 2;
      const rows: string[][] = [];
      while (index < lines.length && isTableRow(lines[index])) {
        rows.push(splitTableRow(lines[index]));
        index++;
      }
      blocks.push({ type: "table", headers, rows });
      continue;
    }

    const headingMatch = line.match(/^\s{0,3}(#{1,3})\s+(.+)$/);
    if (headingMatch) {
      flushParagraph();
      flushList();
      blocks.push({
        type: "heading",
        level: headingMatch[1].length as 1 | 2 | 3,
        text: headingMatch[2].trim(),
      });
      index++;
      continue;
    }

    if (/^\s{0,3}(-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
      flushParagraph();
      flushList();
      blocks.push({ type: "divider" });
      index++;
      continue;
    }

    const quoteMatch = line.match(/^\s{0,3}>\s?(.+)$/);
    if (quoteMatch) {
      flushParagraph();
      flushList();
      blocks.push({ type: "quote", text: quoteMatch[1].trim() });
      index++;
      continue;
    }

    const listMatch = line.match(/^\s*[-*]\s+(.+)$/);
    if (listMatch) {
      flushParagraph();
      list.push(listMatch[1]);
      index++;
      continue;
    }

    if (line.trim() === "") {
      flushParagraph();
      flushList();
      index++;
      continue;
    }

    flushList();
    paragraph.push(line);
    index++;
  }

  flushParagraph();
  flushList();
  return blocks;
}

function isMarkdownTableStart(lines: string[], index: number) {
  return isTableRow(lines[index]) && index + 1 < lines.length && /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/.test(lines[index + 1]);
}

function isTableRow(line: string | undefined) {
  return Boolean(line && line.includes("|") && line.trim().startsWith("|") && line.trim().endsWith("|"));
}

function splitTableRow(line: string) {
  return line
    .trim()
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((cell) => cell.trim());
}

function renderInlineMarkdown(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  const pattern = /(\*\*[^*]+\*\*|`[^`]+`)/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > lastIndex) {
      nodes.push(text.slice(lastIndex, match.index));
    }

    const token = match[0];
    if (token.startsWith("**")) {
      nodes.push(
        <strong key={nodes.length} className="font-semibold">
          {token.slice(2, -2)}
        </strong>,
      );
    } else {
      nodes.push(
        <code key={nodes.length} className="rounded bg-muted px-1 py-0.5 text-[0.85em]">
          {token.slice(1, -1)}
        </code>,
      );
    }

    lastIndex = match.index + token.length;
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex));
  }

  return nodes;
}
