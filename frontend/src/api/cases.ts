import { api } from "./client";
import type { ReplayResult } from "./events";

export interface ReplayCase {
  id: string;
  name: string;
  tags: string;
  provider: string;
  method: string;
  target: string;
  body: string;
  source: string; // seed | captured
  created_at: string;
}

export interface ActiveSession {
  id: string;
  client: string;
  upstream: string;
}

export function fetchCases(): Promise<ReplayCase[]> {
  return api.getJSON<ReplayCase[]>("/api/cases").then((c) => c ?? []);
}

export function fetchActiveSessions(): Promise<ActiveSession[]> {
  return api.getJSON<ActiveSession[]>("/api/active-sessions").then((s) => s ?? []);
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
