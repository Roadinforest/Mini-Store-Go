import { chatStreamSchema, type ChatStreamChunk } from "@/lib/api";

// A truncated stream must be reported as a failure, never retried as a new chat.
export async function consumeChatStream(
  response: Response,
  onChunk: (chunk: ChatStreamChunk) => void | Promise<void>,
  signal?: AbortSignal,
): Promise<void> {
  if (!response.ok || !response.body || !response.headers.get("content-type")?.includes("text/event-stream")) {
    throw new Error("Invalid chat stream response.");
  }
  const reader = response.body.getReader();
  const abort = () => { void reader.cancel().catch(() => {}); };
  signal?.addEventListener("abort", abort, { once: true });
  const decoder = new TextDecoder();
  let buffer = "";
  let completed = false;
  try {
    for (;;) {
      if (signal?.aborted) throw new DOMException("Aborted", "AbortError");
      const { value, done } = await reader.read();
      if (signal?.aborted) throw new DOMException("Aborted", "AbortError");
      buffer += done ? decoder.decode() : decoder.decode(value, { stream: true });
      buffer = buffer.replace(/\r\n/g, "\n");
      let boundary: number;
      while ((boundary = buffer.indexOf("\n\n")) >= 0) {
        const event = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        const data = event.split("\n").filter(line => line.startsWith("data:")).map(line => line.slice(5).trimStart()).join("\n").trim();
        if (!data) continue;
        if (data === "[DONE]") {
          if (!completed) throw new Error("Chat stream ended without a result.");
          return;
        }
        if (completed) throw new Error("Unexpected event after chat completion.");
        const chunk = chatStreamSchema.parse(JSON.parse(data));
        if (chunk.type === "error") throw new Error(chunk.content ?? "Chat stream failed.");
        await onChunk(chunk);
        completed = chunk.type === "complete" || chunk.type === "navigation";
      }
      if (done) throw new Error("Chat stream disconnected before completion.");
    }
  } finally {
    signal?.removeEventListener("abort", abort);
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}
