import { useSessionEvents, useEventDetail } from "@/query/events";
import { useUIStore, type BlockKind } from "@/store/ui";
import type { ContentBlock, EventSummary, Message, TagInfo, Tool, Usage } from "@/api/events";

// --- session event timeline ---
export interface SessionEventsView {
  events: EventSummary[];
  defaultEventId: string | null;
  isLoading: boolean;
  isError: boolean;
}

export function useSessionEventsView(sessionId: string | null): SessionEventsView {
  const { data, isLoading, isError } = useSessionEvents(sessionId);
  const { ordered, defaultEventId } = orderEvents(data ?? []);
  return { events: ordered, defaultEventId, isLoading, isError };
}

// A session-level Turn: one user prompt and everything (http calls + hooks) it
// triggered, up to the next prompt. Index 0 is the pre-prompt session start.
// This is a level ABOVE the request-internal "rounds" in the detail view.
export interface Turn {
  key: string;
  index: number;
  events: EventSummary[];
  httpCount: number;
  hookCount: number;
  startedAt: string;
}

// groupIntoTurns splits a session's events into turns at each UserPromptSubmit
// hook. Returns null when there are no prompt markers (e.g. an http-only or
// codex session) so the caller can fall back to a flat list. Turns are
// chronological; events within a turn are chronological too.
export function groupIntoTurns(events: EventSummary[]): Turn[] | null {
  const hasPrompt = events.some((e) => e.kind === "hook_event" && e.hook_event === "UserPromptSubmit");
  if (!hasPrompt) return null;

  const asc = [...events].sort((a, b) => a.started_at.localeCompare(b.started_at));
  const turns: Turn[] = [];
  let idx = 0;
  let cur: Turn | null = null;
  const start = (e: EventSummary) => {
    cur = { key: `t-${e.id}`, index: idx, events: [], httpCount: 0, hookCount: 0, startedAt: e.started_at };
    turns.push(cur);
  };
  for (const e of asc) {
    if (e.kind === "hook_event" && e.hook_event === "UserPromptSubmit") {
      idx++;
      start(e);
    } else if (!cur) {
      start(e); // session-start group (index 0)
    }
    cur!.events.push(e);
    if (e.kind === "hook_event") cur!.hookCount++;
    else cur!.httpCount++;
  }
  return turns;
}

// --- flow graph (hook events as the first layer; http as a 2nd-layer ref) ---
// The flow's spine is the harness HOOK timeline (orchestration). Each hook node
// carries a reference to the ONE http completion it relates to — so several hook
// events (e.g. the Pre/Post pairs of two tool calls from the same response) point
// to the same request. http itself is not a first-layer node; its detail opens in
// a side sheet. The association uses the strict causal ordering of the lifecycle,
// so it's exact, not a time-based guess:
//   • a tool hook (has tool_call_id) → the completion that PRODUCED it
//     (the last completion at/before the hook)
//   • UserPromptSubmit → the request it TRIGGERED (first completion at/after it)
export interface HttpRef {
  id: string;
  index: number; // 1-based ordinal of the completion within the turn
}

export interface FlowHookNode {
  event: EventSummary;
  httpRef: HttpRef | null;
}

export interface TurnFlow {
  nodes: FlowHookNode[];
}

// buildTurnFlow turns one turn's flat events into the hook-first node list the
// flow graph renders. Pure for testability.
export function buildTurnFlow(events: EventSummary[]): TurnFlow {
  const asc = [...events].sort((a, b) => a.started_at.localeCompare(b.started_at));
  // Only real LLM round-trips are association targets — control/probe requests
  // (is_completion=false, e.g. codex's GET probes) must not be picked as a hook's
  // producing/triggering request, nor consume a request ordinal.
  const completions = asc.filter((e) => e.kind === "http_exchange" && e.is_completion);
  const ordinal = new Map<string, number>();
  let n = 0;
  for (const c of completions) ordinal.set(c.id, ++n);
  const ref = (c: EventSummary): HttpRef => ({ id: c.id, index: ordinal.get(c.id)! });

  const nodes: FlowHookNode[] = [];
  for (const e of asc) {
    if (e.kind !== "hook_event") continue;
    let httpRef: HttpRef | null = null;
    if (e.tool_call_id) {
      let producer: EventSummary | null = null;
      for (const c of completions) {
        if (c.started_at <= e.started_at) producer = c;
        else break;
      }
      if (producer) httpRef = ref(producer);
    } else if (e.hook_event === "UserPromptSubmit") {
      const trig = completions.find((c) => c.started_at >= e.started_at);
      if (trig) httpRef = ref(trig);
    }
    nodes.push({ event: e, httpRef });
  }
  return { nodes };
}

// orderEvents sorts events newest-first and picks the newest completion as the
// default selection. Pure for testability.
export function orderEvents(events: EventSummary[]): {
  ordered: EventSummary[];
  defaultEventId: string | null;
} {
  const ordered = [...events].sort((a, b) => b.started_at.localeCompare(a.started_at));
  const completions = ordered.filter((e) => e.is_completion);
  const pick = completions.length ? completions : ordered;
  return { ordered, defaultEventId: pick.length ? pick[0].id : null };
}

// --- event detail ---
export interface SectionBar {
  name: string;
  tokens: number;
  pct: number;
  color: string;
}

// A round = one user turn and everything that followed it (reasoning, tool
// calls/results, assistant text) up to the next user message. The boundary is a
// structural signal (role === "user"), not a keyword guess. The leading content
// before the first user message is the "preamble" (index 0).
export interface Round {
  key: string;
  index: number; // 0 = preamble
  preview: string;
  toolCalls: number;
  messages: Message[];
}

export interface DetailGroups {
  system: ContentBlock[];
  tools: Tool[];
  requestMessages: Message[];
  requestRounds: Round[];
  responseMessages: Message[];
}

export interface DetailCounts {
  system: number;
  tools: number;
  requestMessages: number;
  responseMessages: number;
}

// CompactionView surfaces option-A's req↔res comparison for a /compact request:
// how big the context that went in was vs the summary that came out, plus the
// summary text itself. null when the event isn't a compaction.
export interface CompactionView {
  contextIn: number; // full input context incl. cache (input + cache_read + cache_creation)
  summaryOut: number; // output tokens = the produced summary
  summaryText: string; // the model's summary (response text blocks)
}

export interface EventDetailView {
  isLoading: boolean;
  isError: boolean;
  found: boolean;
  header: {
    provider: string;
    model: string;
    method: string;
    target: string;
    status: number;
    durationMs: number;
    isCompletion: boolean;
    normalizeError?: string;
  } | null;
  groups: DetailGroups;
  counts: DetailCounts;
  sectionBars: SectionBar[];
  tags: TagInfo[];
  usage: Usage | null;
  sessionId: string;
  // Compaction is decided cross-event (episodes from the API), not from this
  // event's tags/text. These are just the numbers/summary to render IF the
  // event turns out to be a compaction trigger.
  compactionMetrics: CompactionView | null;
  hasRawRequest: boolean;
  hasRawResponse: boolean;
}

const emptyGroups: DetailGroups = {
  system: [],
  tools: [],
  requestMessages: [],
  requestRounds: [],
  responseMessages: [],
};
const emptyCounts: DetailCounts = { system: 0, tools: 0, requestMessages: 0, responseMessages: 0 };

const sectionColor: Record<string, string> = {
  system: "var(--color-accent)",
  tools: "var(--color-toolcall)",
  messages: "var(--color-toolresult)",
};

export function useEventDetailView(eventId: string | null): EventDetailView {
  const { data, isLoading, isError } = useEventDetail(eventId);
  const blockFilter = useUIStore((s) => s.blocks);

  if (!data) {
    return {
      isLoading,
      isError,
      found: false,
      header: null,
      groups: emptyGroups,
      counts: emptyCounts,
      sectionBars: [],
      tags: [],
      usage: null,
      sessionId: "",
      compactionMetrics: null,
      hasRawRequest: false,
      hasRawResponse: false,
    };
  }

  const env = data.normalized;
  const system = env?.request?.system ?? [];
  const tools = env?.request?.tools ?? [];
  const reqMessages = filterMessages(env?.request?.messages ?? [], blockFilter);
  const respMessages = filterMessages(env?.response?.output ?? [], blockFilter);

  return {
    isLoading,
    isError,
    found: true,
    header: {
      provider: data.provider,
      model: data.model,
      method: data.method,
      target: data.target,
      status: data.response_status,
      durationMs: data.duration_ms,
      isCompletion: data.is_completion,
      normalizeError: data.normalize_error,
    },
    groups: {
      system: filterBlocks(system, blockFilter),
      tools,
      requestMessages: reqMessages,
      requestRounds: groupIntoRounds(reqMessages),
      responseMessages: respMessages,
    },
    counts: {
      system: system.length,
      tools: tools.length,
      requestMessages: reqMessages.length,
      responseMessages: respMessages.length,
    },
    sectionBars: buildSectionBars(env?.request?.sections ?? []),
    tags: data.tags,
    usage: env?.response?.usage ?? null,
    sessionId: data.session_id,
    compactionMetrics: extractCompactionMetrics(env),
    hasRawRequest: data.raw_files.some((f) => f.role === "request_body"),
    hasRawResponse: data.raw_files.some((f) => f.role === "response_body"),
  };
}

// extractCompactionMetrics pulls the numbers + summary text the compaction panel
// renders. It does NOT decide whether the event IS a compaction — that's the
// cross-event episode call (useCompactionEpisodes). Returns null when there's no
// response to summarize.
function extractCompactionMetrics(
  env: { response?: { output?: Message[]; usage?: Usage } } | undefined,
): CompactionView | null {
  if (!env?.response) return null;
  const u = env.response.usage;
  const n = (v: unknown): number => (typeof v === "number" ? v : 0);
  const extra = u?.extra ?? {};
  const contextIn = n(u?.input_tokens) + n(extra.cache_read_input_tokens) + n(extra.cache_creation_input_tokens);
  const summaryText = (env.response.output ?? [])
    .flatMap((m) => m.content ?? [])
    .filter((b) => b.type === "text")
    .map((b) => b.text ?? "")
    .join("");
  return { contextIn, summaryOut: n(u?.output_tokens), summaryText };
}

// filterBlocks keeps only blocks whose type is enabled.
export function filterBlocks(blocks: ContentBlock[], enabled: Record<BlockKind, boolean>): ContentBlock[] {
  return blocks.filter((b) => (b.type in enabled ? enabled[b.type as BlockKind] : true));
}

// filterMessages applies the block filter to each message and drops messages
// left with no visible content.
export function filterMessages(messages: Message[], enabled: Record<BlockKind, boolean>): Message[] {
  return messages
    .map((m) => ({ ...m, content: filterBlocks(m.content ?? [], enabled) }))
    .filter((m) => m.content.length > 0);
}

// groupIntoRounds splits a message list into rounds at each user message. The
// run before the first user message becomes the preamble (index 0). This is a
// structural segmentation — honest, not a semantic guess.
export function groupIntoRounds(messages: Message[]): Round[] {
  const rounds: Round[] = [];
  let idx = 0;

  const startRound = (preview: string) => {
    const r: Round = {
      key: idx === 0 ? "preamble" : `round-${idx}`,
      index: idx,
      preview,
      toolCalls: 0,
      messages: [],
    };
    rounds.push(r);
    return r;
  };

  let current: Round | null = null;
  for (const m of messages) {
    if (m.role === "user") {
      idx += 1;
      current = startRound(firstLine(m));
    } else if (!current) {
      current = startRound(""); // preamble
    }
    current.messages.push(m);
    current.toolCalls += (m.content ?? []).filter((b) => b.type === "tool_call").length;
  }
  return rounds.filter((r) => r.messages.length > 0);
}

function firstLine(m: Message): string {
  const text = (m.content ?? []).find((b) => b.type === "text")?.text ?? "";
  return text.split("\n")[0].slice(0, 80);
}

export function buildSectionBars(sections: { name: string; approx_tokens: number }[]): SectionBar[] {
  const total = sections.reduce((sum, s) => sum + s.approx_tokens, 0) || 1;
  return sections.map((s) => ({
    name: s.name,
    tokens: s.approx_tokens,
    pct: (s.approx_tokens / total) * 100,
    color: sectionColor[s.name] ?? "var(--color-muted-foreground)",
  }));
}
