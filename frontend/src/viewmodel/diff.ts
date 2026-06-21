import type { ContentBlock, EventDetail, Message } from "@/api/events";

export type DiffKind = "unchanged" | "added" | "removed";

export interface DiffOp {
  kind: DiffKind;
  message: Message;
}

// DiffSummary classifies what the harness did to the context this turn, so the
// reader gets the gist without scanning every row. compactionSuspected is a
// heuristic (a long removed run) — shown as 疑似, never asserted (next.md 3.3).
export interface DiffSummary {
  toolResultsAdded: number;
  systemChanged: boolean;
  compactionSuspected: boolean;
}

export function summarizeDiff(ops: DiffOp[]): DiffSummary {
  let toolResultsAdded = 0;
  let systemChanged = false;
  let maxRemovedRun = 0;
  let run = 0;
  for (const o of ops) {
    if (o.kind === "removed") {
      run += 1;
      maxRemovedRun = Math.max(maxRemovedRun, run);
    } else {
      run = 0;
    }
    if (o.kind !== "unchanged" && o.message.role === "system") systemChanged = true;
    if (
      o.kind === "added" &&
      (o.message.role === "tool" || (o.message.content ?? []).some((b) => b.type === "tool_result"))
    ) {
      toolResultsAdded += 1;
    }
  }
  return { toolResultsAdded, systemChanged, compactionSuspected: maxRemovedRun >= 3 };
}

// requestSequence is the comparable input sequence of a completion: its system
// (as one message) followed by the request messages.
export function requestSequence(detail: EventDetail | undefined): Message[] {
  const req = detail?.normalized?.request;
  if (!req) return [];
  const seq: Message[] = [];
  if (req.system && req.system.length) seq.push({ role: "system", content: req.system });
  if (req.messages) seq.push(...req.messages);
  return seq;
}

// messageKey is a stable identity for a message, used to align two sequences.
export function messageKey(m: Message): string {
  const blocks = (m.content ?? []).map(blockKey).join("|");
  return `${m.role}#${blocks}`;
}

function blockKey(b: ContentBlock): string {
  switch (b.type) {
    case "text":
    case "reasoning":
      return `${b.type}:${b.text ?? ""}`;
    case "tool_call":
      return `tc:${b.tool_call?.id ?? ""}:${b.tool_call?.name ?? ""}:${JSON.stringify(b.tool_call?.arguments ?? null)}`;
    case "tool_result":
      return `tr:${b.tool_result?.tool_call_id ?? ""}:${(b.tool_result?.content ?? []).map((c) => c.text ?? "").join("")}`;
    default:
      return b.type;
  }
}

// diffMessages aligns two message sequences via LCS and returns the ordered
// add/remove/unchanged operations. Consecutive requests share a long common
// prefix, so this cleanly surfaces what a turn appended — and a large removed
// run replaced by a summary is the signature of compaction.
export function diffMessages(left: Message[], right: Message[]): DiffOp[] {
  const a = left.map(messageKey);
  const b = right.map(messageKey);
  const n = a.length;
  const m = b.length;

  // dp[i][j] = LCS length of a[i:], b[j:]
  const dp: Int32Array[] = Array.from({ length: n + 1 }, () => new Int32Array(m + 1));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = a[i] === b[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }

  const ops: DiffOp[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      ops.push({ kind: "unchanged", message: right[j] });
      i++;
      j++;
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      ops.push({ kind: "removed", message: left[i] });
      i++;
    } else {
      ops.push({ kind: "added", message: right[j] });
      j++;
    }
  }
  while (i < n) ops.push({ kind: "removed", message: left[i++] });
  while (j < m) ops.push({ kind: "added", message: right[j++] });
  return ops;
}
