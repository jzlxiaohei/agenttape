import { describe, it, expect } from "vitest";
import { buildSectionBars, filterBlocks, filterMessages, groupIntoRounds, groupIntoTurns, orderEvents } from "./detail";
import type { BlockKind } from "@/store/ui";
import type { ContentBlock, EventSummary, Message } from "@/api/events";

function ev(id: string, started_at: string, is_completion: boolean): EventSummary {
  return {
    id,
    kind: "http_exchange",
    started_at,
    method: "POST",
    target: "",
    provider: "",
    model: "",
    is_completion,
    response_status: 200,
    total_tokens: 0,
    hook_event: "",
    tool_call_id: "",
  };
}

function hookEv(id: string, started_at: string, name: string): EventSummary {
  return { ...ev(id, started_at, false), kind: "hook_event", hook_event: name };
}

describe("groupIntoTurns", () => {
  it("returns null when there are no prompt markers (fall back to flat)", () => {
    expect(groupIntoTurns([ev("a", "2026-06-19T10:00:00Z", true)])).toBeNull();
  });

  it("splits at UserPromptSubmit with a pre-prompt session group", () => {
    const events = [
      hookEv("s", "2026-06-19T10:00:00Z", "SessionStart"),
      hookEv("p1", "2026-06-19T10:01:00Z", "UserPromptSubmit"),
      ev("h1", "2026-06-19T10:01:30Z", true),
      hookEv("pt", "2026-06-19T10:01:40Z", "PreToolUse"),
      hookEv("p2", "2026-06-19T10:05:00Z", "UserPromptSubmit"),
      ev("h2", "2026-06-19T10:05:30Z", true),
    ];
    const turns = groupIntoTurns(events)!;
    expect(turns.map((t) => t.index)).toEqual([0, 1, 2]);
    expect(turns[0].events.map((e) => e.id)).toEqual(["s"]); // session start
    expect(turns[1].httpCount).toBe(1);
    expect(turns[1].hookCount).toBe(2); // UserPromptSubmit + PreToolUse
    expect(turns[2].events.map((e) => e.id)).toEqual(["p2", "h2"]);
  });
});

describe("orderEvents", () => {
  it("sorts newest-first and defaults to the newest completion", () => {
    const events = [
      ev("a", "2026-06-18T10:00:00Z", true),
      ev("probe", "2026-06-18T10:05:00Z", false),
      ev("b", "2026-06-18T10:02:00Z", true),
    ];
    const { ordered, defaultEventId } = orderEvents(events);
    expect(ordered.map((e) => e.id)).toEqual(["probe", "b", "a"]);
    expect(defaultEventId).toBe("b"); // newest completion, not the newer probe
  });
});

const allOn: Record<BlockKind, boolean> = {
  text: true,
  reasoning: true,
  tool_call: true,
  tool_result: true,
};

describe("buildSectionBars", () => {
  it("computes percentages that sum to 100", () => {
    const bars = buildSectionBars([
      { name: "system", approx_tokens: 10 },
      { name: "tools", approx_tokens: 30 },
      { name: "messages", approx_tokens: 60 },
    ]);
    expect(bars.map((b) => Math.round(b.pct))).toEqual([10, 30, 60]);
    expect(bars.reduce((s, b) => s + b.pct, 0)).toBeCloseTo(100);
  });

  it("does not divide by zero when all sections are empty", () => {
    const bars = buildSectionBars([{ name: "system", approx_tokens: 0 }]);
    expect(bars[0].pct).toBe(0);
  });
});

describe("filterBlocks", () => {
  it("keeps only enabled block types", () => {
    const blocks: ContentBlock[] = [
      { type: "text", text: "a" },
      { type: "reasoning", text: "r" },
      { type: "tool_call", tool_call: { name: "x" } },
    ];
    const out = filterBlocks(blocks, { ...allOn, reasoning: false });
    expect(out.map((b) => b.type)).toEqual(["text", "tool_call"]);
  });
});

describe("groupIntoRounds", () => {
  it("splits at each user message and keeps a leading preamble", () => {
    const msgs: Message[] = [
      { role: "system", content: [{ type: "text", text: "sys" }] },
      { role: "user", content: [{ type: "text", text: "first ask\nmore" }] },
      { role: "assistant", content: [{ type: "tool_call", tool_call: { name: "Bash" } }] },
      { role: "tool", content: [{ type: "tool_result", tool_result: {} }] },
      { role: "user", content: [{ type: "text", text: "second ask" }] },
      { role: "assistant", content: [{ type: "text", text: "done" }] },
    ];
    const rounds = groupIntoRounds(msgs);

    expect(rounds.map((r) => r.index)).toEqual([0, 1, 2]);
    expect(rounds[0].key).toBe("preamble");
    expect(rounds[1].preview).toBe("first ask"); // first line only
    expect(rounds[1].toolCalls).toBe(1);
    expect(rounds[1].messages).toHaveLength(3); // user + assistant + tool
    expect(rounds[2].messages).toHaveLength(2);
  });

  it("has no preamble when the first message is a user message", () => {
    const rounds = groupIntoRounds([{ role: "user", content: [{ type: "text", text: "hi" }] }]);
    expect(rounds).toHaveLength(1);
    expect(rounds[0].index).toBe(1);
  });
});

describe("filterMessages", () => {
  it("drops messages left with no visible blocks after filtering", () => {
    const messages: Message[] = [
      { role: "assistant", content: [{ type: "reasoning", text: "think" }] },
      { role: "user", content: [{ type: "text", text: "hi" }] },
    ];
    const out = filterMessages(messages, { ...allOn, reasoning: false });
    expect(out).toHaveLength(1);
    expect(out[0].role).toBe("user");
  });
});
