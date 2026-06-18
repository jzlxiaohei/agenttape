import { describe, it, expect } from "vitest";
import { diffMessages, messageKey } from "./diff";
import type { Message } from "@/api/events";

const msg = (role: string, text: string): Message => ({ role, content: [{ type: "text", text }] });

describe("diffMessages", () => {
  it("marks appended messages as added and shared prefix as unchanged", () => {
    const left = [msg("user", "a"), msg("assistant", "b")];
    const right = [msg("user", "a"), msg("assistant", "b"), msg("user", "c")];
    const ops = diffMessages(left, right);
    expect(ops.map((o) => o.kind)).toEqual(["unchanged", "unchanged", "added"]);
    expect(ops[2].message.content?.[0].text).toBe("c");
  });

  it("marks dropped messages as removed (e.g. compaction replacing history)", () => {
    const left = [msg("user", "a"), msg("assistant", "b"), msg("tool", "c")];
    const right = [msg("system", "summary"), msg("user", "d")];
    const ops = diffMessages(left, right);
    const kinds = ops.map((o) => o.kind);
    expect(kinds.filter((k) => k === "removed").length).toBe(3);
    expect(kinds.filter((k) => k === "added").length).toBe(2);
  });

  it("messageKey distinguishes role and content", () => {
    expect(messageKey(msg("user", "x"))).not.toBe(messageKey(msg("assistant", "x")));
    expect(messageKey(msg("user", "x"))).toBe(messageKey(msg("user", "x")));
  });
});
