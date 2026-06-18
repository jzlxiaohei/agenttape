import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { TagList } from "./TagList";
import type { TagInfo } from "@/api/events";

describe("TagList", () => {
  it("marks suspected tags as uncertain and exposes evidence", () => {
    const tags: TagInfo[] = [
      { tag: "tool_call", confidence: 1, suspected: false, source: "structural", evidence: "" },
      {
        tag: "compaction",
        confidence: 0.5,
        suspected: true,
        source: "heuristic",
        evidence: "matched marker: continued from a previous conversation",
      },
    ];
    render(<TagList tags={tags} />);

    // suspected marker present
    expect(screen.getByText(/suspected/i)).toBeInTheDocument();
    // evidence is exposed via title attribute (tooltip)
    const compaction = screen.getByText("compaction");
    expect(compaction.closest("span")).toHaveAttribute("title", expect.stringContaining("marker"));
  });

  it("renders nothing when there are no tags", () => {
    const { container } = render(<TagList tags={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
