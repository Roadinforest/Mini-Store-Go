import { MessageCircle, SendHorizontal, X } from "lucide-react";
import { FormEvent, useEffect, useRef, useState } from "react";
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
    setMessages(nextMessages);
    setInput("");
    setError(null);
    setIsLoading(true);
    let placeholderAdded = false;

    try {
      const response = await createChatStream(nextMessages);
      if (!response.ok || !response.body) {
        throw new Error("stream unavailable");
      }

      setMessages((current) => [...current, { role: "assistant", content: "" }]);
      placeholderAdded = true;
      await consumeStream(response, setMessages);
    } catch {
      const fallback = await sendChat(nextMessages);
      if (!fallback.success || !fallback.data) {
        setError(fallback.message || "智能助手暂时不可用，请稍后再试。");
        setMessages((current) =>
          placeholderAdded
            ? replaceLastAssistant(current, "智能助手暂时不可用，请稍后再试。")
            : [...current, { role: "assistant", content: "智能助手暂时不可用，请稍后再试。" }],
        );
      } else {
        setMessages((current) => (placeholderAdded ? replaceLastAssistant(current, fallback.data!.content) : [...current, fallback.data!]));
      }
    } finally {
      setIsLoading(false);
    }
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
                  {message.content || (isLoading && index === messages.length - 1 ? "正在思考..." : "")}
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

async function consumeStream(
  response: Response,
  setMessages: (value: ChatMessage[] | ((current: ChatMessage[]) => ChatMessage[])) => void,
) {
  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error("no response body");
  }

  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const chunks = buffer.split("\n\n");
    buffer = chunks.pop() ?? "";

    for (const chunk of chunks) {
      const line = chunk
        .split("\n")
        .find((item) => item.startsWith("data: "));
      if (!line) continue;

      const payload = line.slice(6);
      if (payload === "[DONE]") continue;

      const data = JSON.parse(payload) as ChatStreamChunk;
      applyStreamChunk(data, setMessages);
    }
  }
}

function applyStreamChunk(
  chunk: ChatStreamChunk,
  setMessages: (value: ChatMessage[] | ((current: ChatMessage[]) => ChatMessage[])) => void,
) {
  if (chunk.type === "error") {
    setMessages((current) => replaceLastAssistant(current, chunk.content || "智能助手暂时不可用，请稍后再试。"));
    return;
  }

  if (chunk.type === "partial") {
    setMessages((current) => appendToLastAssistant(current, chunk.content || ""));
    return;
  }

  if (chunk.type === "complete" || chunk.type === "thinking") {
    setMessages((current) => replaceLastAssistant(current, chunk.content || ""));
  }
}

function replaceLastAssistant(messages: ChatMessage[], content: string) {
  const next = [...messages];
  const lastIndex = next.length - 1;
  if (lastIndex < 0 || next[lastIndex]?.role !== "assistant") {
    next.push({ role: "assistant", content });
    return next;
  }

  next[lastIndex] = {
    ...next[lastIndex],
    content,
  };
  return next;
}

function appendToLastAssistant(messages: ChatMessage[], content: string) {
  const next = [...messages];
  const lastIndex = next.length - 1;
  if (lastIndex < 0 || next[lastIndex]?.role !== "assistant") {
    next.push({ role: "assistant", content });
    return next;
  }

  next[lastIndex] = {
    ...next[lastIndex],
    content: `${next[lastIndex].content}${content}`,
  };
  return next;
}
