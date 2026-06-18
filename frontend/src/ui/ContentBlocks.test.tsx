import { describe, it, expect, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ContentBlocks } from "./ContentBlocks";
import { useUIStore } from "@/store/ui";
import type { ContentBlock } from "@/api/events";

afterEach(() => useUIStore.setState({ renderMarkdown: false }));

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
    render(<ContentBlocks blocks={blocks} />);

    expect(screen.getByText("plain answer")).toBeInTheDocument();
    expect(screen.getByText("thinking hard")).toBeInTheDocument();
    // tool name + json args
    expect(screen.getByText("Bash")).toBeInTheDocument();
    expect(screen.getByText(/"cmd": "ls"/)).toBeInTheDocument();
    // nested tool_result content
    expect(screen.getByText("result text")).toBeInTheDocument();
  });

  it("renders nothing for empty blocks", () => {
    const { container } = render(<ContentBlocks blocks={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders markdown (lazy) when the toggle is on, keeping raw as fallback", async () => {
    useUIStore.setState({ renderMarkdown: true });
    render(<ContentBlocks blocks={[{ type: "text", text: "**bold** word" }]} />);
    const strong = await screen.findByText("bold");
    expect(strong.tagName).toBe("STRONG");
  });
});
