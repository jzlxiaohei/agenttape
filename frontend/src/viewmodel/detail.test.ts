import { describe, it, expect } from "vitest";
import { buildSectionBars, buildTurnFlow, filterBlocks, filterMessages, groupIntoRounds, groupIntoTurns, orderEvents } from "./detail";
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
    tool_name: "",
  };
}

function hookEv(id: string, started_at: string, name: string): EventSummary {
  return { ...ev(id, started_at, false), kind: "hook_event", hook_event: name };
}

function toolHook(id: string, at: string, name: string, callId: string, toolName = ""): EventSummary {
  return { ...hookEv(id, at, name), tool_call_id: callId, tool_name: toolName };
}

describe("buildTurnFlow", () => {
  it("makes hooks the first layer; several tool hooks point to the producing completion", () => {
    // completion #1 produces call_A + call_B → their Pre/Post hooks all ref #1
    const events = [
      ev("c1", "2026-06-19T10:00:00Z", true),
      toolHook("preA", "2026-06-19T10:00:01Z", "PreToolUse", "call_A", "shell"),
      toolHook("postA", "2026-06-19T10:00:02Z", "PostToolUse", "call_A", "shell"),
      toolHook("preB", "2026-06-19T10:00:02Z", "PreToolUse", "call_B", "read"),
      ev("c2", "2026-06-19T10:00:05Z", true),
    ];
    const flow = buildTurnFlow(events);
    // only hooks are nodes; http is not a first-layer node
    expect(flow.nodes.map((nd) => nd.event.id)).toEqual(["preA", "postA", "preB"]);
    expect(flow.nodes.every((nd) => nd.httpRef?.index === 1)).toBe(true);
  });

  it("links UserPromptSubmit to the request it triggered; id-less Stop has no ref", () => {
    const events = [
      hookEv("u", "2026-06-19T10:00:00Z", "UserPromptSubmit"),
      ev("c1", "2026-06-19T10:00:01Z", true),
      hookEv("stop", "2026-06-19T10:00:09Z", "Stop"),
    ];
    const flow = buildTurnFlow(events);
    expect(flow.nodes.find((nd) => nd.event.id === "u")?.httpRef?.index).toBe(1);
    expect(flow.nodes.find((nd) => nd.event.id === "stop")?.httpRef).toBeNull();
  });

  it("ignores control/probe requests (is_completion=false) when associating", () => {
    const events = [
      ev("c1", "2026-06-19T10:00:01Z", true), // real completion #1
      ev("probe", "2026-06-19T10:00:02Z", false), // probe after it — must be ignored
      toolHook("pre", "2026-06-19T10:00:03Z", "PreToolUse", "call_A", "shell"),
    ];
    const flow = buildTurnFlow(events);
    const node = flow.nodes.find((nd) => nd.event.id === "pre");
    expect(node?.httpRef?.id).toBe("c1"); // not "probe"
    expect(node?.httpRef?.index).toBe(1);
  });
});

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
