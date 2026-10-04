import { describe, expect, it } from "vitest";
import { consumeChatStream } from "./chat-stream";

function response(parts: string[]) {
  const encoder = new TextEncoder();
  return new Response(new ReadableStream({start(controller){parts.forEach(part => controller.enqueue(encoder.encode(part))); controller.close();}}),{headers:{"Content-Type":"text/event-stream"}});
}
describe("chat stream transport", () => {
  it("parses split SSE events with CRLF and tool results", async () => {
    const events: string[]=[];
    await consumeChatStream(response(['data: {"type":"tool_call"}\r','\n\r\ndata: {"type":"tool_result","toolName":"search_products"}\r\n\r\n','data: {"type":"partial","content":"你好"}\n\ndata: {"type":"complete","content":"你好"}\n\ndata: [DONE]\n\n']),event=>{events.push(event.type)});
    expect(events).toEqual(["tool_call","tool_result","partial","complete"]);
  });
  it.each([
    'data: {"type":"partial","content":"partial"}\n\n',
    'data: [DONE]\n\n',
    'data: {"type":"error","content":"upstream failed"}\n\ndata: [DONE]\n\n',
    'data: {"type":"invented"}\n\n',
  ])("rejects truncated, error and malformed streams", async payload => {
    await expect(consumeChatStream(response([payload]),()=>{})).rejects.toThrow();
  });
  it("cancels and releases a pending reader on abort", async () => {
    let canceled=false;
    const stream=new ReadableStream({cancel(){canceled=true;}});
    const controller=new AbortController();
    const pending=consumeChatStream(new Response(stream,{headers:{"Content-Type":"text/event-stream"}}),()=>{},controller.signal);
    controller.abort();
    await expect(pending).rejects.toMatchObject({name:"AbortError"});
    expect(canceled).toBe(true);
    expect(stream.locked).toBe(false);
  });
});
