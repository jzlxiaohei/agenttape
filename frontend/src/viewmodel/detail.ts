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

export interface EventDetailView {
  isLoading: boolean;
  isError: boolean;
  found: boolean;
  header: {
    provider: string;
    model: string;
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
    hasRawRequest: data.raw_files.some((f) => f.role === "request_body"),
    hasRawResponse: data.raw_files.some((f) => f.role === "response_body"),
  };
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
