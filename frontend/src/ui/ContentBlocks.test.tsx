import { describe, it, expect, afterEach, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ContentBlocks } from "./ContentBlocks";
import { useUIStore } from "@/store/ui";
import type { ContentBlock } from "@/api/events";

vi.mock("./Markdown", () => ({
  default: ({ text }: { text: string }) => (
    <div data-testid="markdown">
      {text === "**bold** word" ? (
        <>
          <strong>bold</strong> word
        </>
      ) : (
        text
      )}
    </div>
  ),
}));

afterEach(() => {
  act(() => useUIStore.setState({ renderMarkdown: false }));
});

// ContentBlocks reads navigation state via the route hook, so it needs a Router.
const renderCB = (blocks: ContentBlock[]) =>
  render(<ContentBlocks blocks={blocks} />, { wrapper: MemoryRouter });

describe("ContentBlocks", () => {
  it("renders text, reasoning, tool_call, and tool_result distinctly by type", () => {
    const blocks: ContentBlock[] = [
      { type: "text", text: "plain answer" },
      { type: "reasoning", text: "thinking hard" },
      { type: "tool_call", tool_call: { name: "Bash", arguments: { cmd: "ls" } } },
      {
        type: "tool_result",
        tool_result: { content: [{ type: "text", text: "result text" }], is_error: false },
      },
    ];
    renderCB(blocks);

    expect(screen.getByText("plain answer")).toBeInTheDocument();
    expect(screen.getByText("thinking hard")).toBeInTheDocument();
    // tool name + json args
    expect(screen.getByText("Bash")).toBeInTheDocument();
    expect(screen.getByText(/"cmd": "ls"/)).toBeInTheDocument();
    // nested tool_result content
    expect(screen.getByText("result text")).toBeInTheDocument();
  });

  it("renders nothing for empty blocks", () => {
    const { container } = renderCB([]);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders markdown (lazy) when the toggle is on, keeping raw as fallback", async () => {
    act(() => useUIStore.setState({ renderMarkdown: true }));
    renderCB([{ type: "text", text: "**bold** word" }]);
    const strong = await screen.findByText("bold", {}, { timeout: 3000 });
    expect(strong.tagName).toBe("STRONG");
  });
});
