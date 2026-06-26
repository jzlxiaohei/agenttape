import { api } from "./client";
import type { ReplayResult } from "./events";

export interface ReplayCase {
  id: string;
  name: string;
  tags: string;
  provider: string;
  method: string;
  target: string;
  endpoint: string;
  body: string;
  source: string; // seed | captured | snapshot | manual
  created_at: string;
}

export interface ActiveSession {
  id: string;
  client: string;
  upstream: string;
  provider: string;
  credential_kind: "key" | "subscription";
  // needs_key: a key-mode session restored after a restart whose real key was never
  // persisted. It routes but would 401 until the key is re-supplied.
  needs_key?: boolean;
}

export function fetchCases(): Promise<ReplayCase[]> {
  return api.getJSON<ReplayCase[]>("/api/cases").then((c) => c ?? []);
}

export function fetchActiveSessions(): Promise<ActiveSession[]> {
  return api.getJSON<ActiveSession[]>("/api/active-sessions").then((s) => s ?? []);
}

// closeActiveSession forgets a live session from the proxy registry (drops its
// in-memory creds). It does not kill the agent process in the user's terminal.
export function closeActiveSession(id: string): Promise<void> {
  return api.del(`/api/active-sessions/${id}`);
}

// reenterSessionKey re-supplies the API key for a key-mode session that lost it on a
// agenttape restart. The key goes only into server memory (never to disk); the still-
// running agent resumes on its next request.
export function reenterSessionKey(id: string, apiKey: string): Promise<{ ok: boolean }> {
  return api.postJSON<{ ok: boolean }>(`/api/active-sessions/${id}/key`, { api_key: apiKey });
}

export function runCase(id: string, sessionId: string, body?: string): Promise<ReplayResult> {
  return api.postJSON<ReplayResult>(`/api/cases/${id}/run`, {
    session_id: sessionId,
    ...(body === undefined ? {} : { body }),
  });
}

export function addCase(eventId: string, name?: string, tags?: string): Promise<ReplayCase> {
  return api.postJSON<ReplayCase>("/api/cases", { event_id: eventId, name: name ?? "", tags: tags ?? "" });
}

export function createCase(input: {
  name: string;
  tags?: string;
  provider: string;
  method?: string;
  target?: string;
  endpoint: string;
  body: string;
}): Promise<ReplayCase> {
  return api.postJSON<ReplayCase>("/api/cases", {
    ...input,
    method: input.method ?? "POST",
    tags: input.tags ?? "",
    target: input.target ?? "",
  });
}

// snapshotCase saves an edited body as a new case derived from `id` (never mutates
// the original). Returns the freshly created snapshot.
export function snapshotCase(id: string, body: string, name?: string): Promise<ReplayCase> {
  return api.postJSON<ReplayCase>(`/api/cases/${id}/snapshot`, { body, name: name ?? "" });
}

export function deleteCase(id: string): Promise<void> {
  return api.del(`/api/cases/${id}`);
}

// overwriteCase saves an edited body back onto the SAME case (in-place, by id),
// unlike snapshot which forks a new one. Irreversible.
export function overwriteCase(id: string, body: string): Promise<ReplayCase> {
  return api.postJSON<ReplayCase>(`/api/cases/${id}/overwrite`, { body });
}

export type CurlMode = "proxy" | "direct";

export interface CaseCurl {
  curl: string;
  has_auth: boolean;
  revealed: boolean;
  credential_kind?: "key" | "subscription";
}

// caseCurl builds a copy-pasteable curl for a case bound to a live session.
// proxy mode carries no secret; direct mode embeds auth (masked unless reveal).
export function caseCurl(
  id: string,
  input: { sessionId: string; mode: CurlMode; reveal?: boolean; body?: string },
): Promise<CaseCurl> {
  return api.postJSON<CaseCurl>(`/api/cases/${id}/curl`, {
    session_id: input.sessionId,
    mode: input.mode,
    reveal: input.reveal ?? false,
    ...(input.body === undefined ? {} : { body: input.body }),
  });
}
