import { api } from "./client";

// DTO mirrors store.SessionSummary from the Go API.
export interface SessionDTO {
  id: string;
  client: string;
  upstream: string;
  title: string; // derived from the first user prompt; may be empty
  started_at: string;
  event_count: number;
  hook_count: number; // 0 with traffic = HTTP-only capture (env "run yourself", no hooks)
}

// httpOnlySession reports whether a session captured traffic but no harness hooks
// (the lightweight env launch path). New/empty sessions return false (unknown yet).
export function httpOnlySession(s: { event_count: number; hook_count: number }): boolean {
  return s.event_count > 0 && s.hook_count === 0;
}

export function fetchSessions(): Promise<SessionDTO[]> {
  return api.getJSON<SessionDTO[]>("/api/sessions").then((s) => s ?? []);
}

// deleteSession permanently removes a captured session and all its events/raw bytes.
export function deleteSession(id: string): Promise<void> {
  return api.del(`/api/sessions/${id}`);
}

// renameSession sets a user-chosen name, overriding the auto-derived title. Empty
// clears it.
export function renameSession(id: string, title: string): Promise<void> {
  return api.patchJSON(`/api/sessions/${id}`, { title });
}
