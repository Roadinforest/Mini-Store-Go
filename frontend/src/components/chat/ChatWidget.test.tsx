import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { createChatStream, sendChat } from "@/lib/api";
import { ChatWidget } from "./ChatWidget";
vi.mock("@/lib/api", async importOriginal => ({
  ...await importOriginal<typeof import("@/lib/api")>(),
  createChatStream:vi.fn(),sendChat:vi.fn(),
}));
beforeEach(()=>{HTMLElement.prototype.scrollTo=vi.fn();});
afterEach(()=>{cleanup();vi.resetAllMocks();});
function ask() {
  render(<ChatWidget/>);
  fireEvent.click(screen.getByLabelText("Open chat assistant"));
  fireEvent.change(screen.getByPlaceholderText("输入你的问题..."),{target:{value:"phone"}});
  fireEvent.click(screen.getByLabelText("Send message"));
}
it.each([500,503])("does not regenerate when the stream endpoint fails (%s)",async status=>{
  vi.mocked(createChatStream).mockResolvedValue(new Response("",{status}));
  ask();
  await waitFor(()=>expect(screen.getByText("智能助手暂时不可用，请稍后再试。")).toBeTruthy());
  expect(sendChat).not.toHaveBeenCalled();
});
it("only falls back when the stream endpoint is unavailable",async()=>{
  vi.mocked(createChatStream).mockResolvedValue(new Response("",{status:404}));
  vi.mocked(sendChat).mockResolvedValue({success:true,message:"OK",data:{role:"assistant",content:"Found phone"}});
  ask();
  await waitFor(()=>expect(screen.getByText("Found phone")).toBeTruthy());
  expect(sendChat).toHaveBeenCalledOnce();
});
