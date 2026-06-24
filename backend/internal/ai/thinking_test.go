package ai

import "testing"

func TestVisibleContentRemovesThinkBlocks(t *testing.T) {
	input := "<think>plan privately</think>你好\n<think>more notes</think>查到了 Apple 产品。"
	got := VisibleContent(input)
	want := "你好\n查到了 Apple 产品。"

	if got != want {
		t.Fatalf("VisibleContent() = %q, want %q", got, want)
	}
}

func TestVisibleContentRemovesUnclosedThinkBlock(t *testing.T) {
	input := "可见内容\n<think>still thinking"
	got := VisibleContent(input)
	want := "可见内容"

	if got != want {
		t.Fatalf("VisibleContent() = %q, want %q", got, want)
	}
}
