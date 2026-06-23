import { api } from "./client";

// --- normalized envelope (mirrors Go internal/normalize) ---
export interface ToolCall {
  id?: string;
  name?: string;
  arguments?: unknown;
}
export interface ToolResult {
  tool_call_id?: string;
  content?: ContentBlock[];
  is_error?: boolean;
}
export interface ContentBlock {
  type: "text" | "reasoning" | "tool_call" | "tool_result" | "image" | "unknown";
  text?: string;
  tool_call?: ToolCall;
  tool_result?: ToolResult;
}
export interface Message {
  role: string;
  content?: ContentBlock[];
}
export interface Tool {
  name?: string;
  description?: string;
  input_schema?: unknown;
}
export interface SectionStat {
  name: string;
  bytes: number;
  approx_tokens: number;
}
export interface Usage {
  input_tokens?: number;
  output_tokens?: number;
  total_tokens?: number;
  extra?: Record<string, unknown>;
}
export interface NormalizedEnvelope {
  provider: { name: string; model?: string; endpoint?: string; wire_api?: string };
  request?: {
    system?: ContentBlock[];
    messages?: Message[];
    tools?: Tool[];
    parameters?: Record<string, unknown>;
    sections?: SectionStat[];
  };
  response?: {
    output?: Message[];
    final_text?: string;
    tool_calls?: ToolCall[];
    usage?: Usage;
    stop_reason?: string;
  };
  signals?: { tag: string; confidence: number; suspected?: boolean; evidence?: string }[];
}

// --- API DTOs (mirror Go store) ---
export interface EventSummary {
  id: string;
  kind: string;
  started_at: string;
  method: string;
  target: string;
  provider: string;
  model: string;
  is_completion: boolean;
  response_status: number;
  total_tokens: number;
  hook_event: string;
  tool_call_id: string;
  tool_name: string;
}
export interface TagInfo {
  tag: string;
  confidence: number;
  suspected: boolean;
  source: string;
  evidence: string;
}
export interface RawFileInfo {
  role: string;
  media_type: string;
  size_bytes: number;
}
export interface EventDetail {
  id: string;
  kind: string;
  session_id: string;
  started_at: string;
  completed_at: string;
  duration_ms: number;
  method: string;
  target: string;
  response_status: number;
  provider: string;
  model: string;
  is_completion: boolean;
  normalize_error?: string;
  normalized?: NormalizedEnvelope;
  tags: TagInfo[];
  raw_files: RawFileInfo[];
  // hook-only
  runtime?: string;
  event_name?: string;
  tool_call_id?: string;
}

export function fetchSessionEvents(sessionId: string): Promise<EventSummary[]> {
  return api.getJSON<EventSummary[]>(`/api/sessions/${sessionId}/events`).then((e) => e ?? []);
}

export function fetchEventDetail(eventId: string): Promise<EventDetail> {
  return api.getJSON<EventDetail>(`/api/events/${eventId}`);
}

// --- compaction episodes (cross-event, graded) ---
export type CompactionGrade = "confirmed" | "strong_suspected" | "weak_suspected";

export interface CompactionEpisode {
  grade: CompactionGrade;
  before_event: string; // A — request whose response is the summary
  after_event: string; // B — first request after the boundary
  evidence: string;
  context_in: number;
  summary_out: number;
}

export function fetchCompactionEpisodes(sessionId: string): Promise<CompactionEpisode[]> {
  return api.getJSON<CompactionEpisode[]>(`/api/sessions/${sessionId}/compaction-episodes`).then((e) => e ?? []);
}

export function rawUrl(eventId: string, role: string): string {
  return `/api/events/${eventId}/raw/${role}`;
}

export function fetchRaw(eventId: string, role: string): Promise<string> {
  return api.getText(rawUrl(eventId, role));
}

// --- replay ---
export interface ReplayResult {
  status: number;
  duration_ms: number;
  normalized?: NormalizedEnvelope;
  normalize_error?: string;
  response_body?: string; // raw upstream body (capped), so non-2xx errors are visible
  truncated?: boolean;
}

// replayEvent re-sends a captured completion to upstream (optionally with an
// edited body) and returns the freshly normalized result. body undefined =
// resend the original verbatim.
export function replayEvent(eventId: string, body?: string): Promise<ReplayResult> {
  return api.postJSON<ReplayResult>(`/api/events/${eventId}/replay`, body === undefined ? {} : { body });
}
