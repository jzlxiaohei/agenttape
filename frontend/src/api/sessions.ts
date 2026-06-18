import { api } from "./client";

// DTO mirrors store.SessionSummary from the Go API.
export interface SessionDTO {
  id: string;
  client: string;
  upstream: string;
  started_at: string;
  event_count: number;
}

export function fetchSessions(): Promise<SessionDTO[]> {
  return api.getJSON<SessionDTO[]>("/api/sessions").then((s) => s ?? []);
}
