import { api } from "./client";

// One configurable hook event for a client: whether tracelab wires it on launch,
// and whether it is a built-in default (seed) or user-added.
export interface HookEventDef {
  client: string;
  event: string;
  enabled: boolean;
  source: string; // seed | user
}

export function fetchHookEvents(): Promise<HookEventDef[]> {
  return api.getJSON<HookEventDef[]>("/api/hook-events").then((d) => d ?? []);
}

export function addHookEvent(client: string, event: string): Promise<void> {
  return api.postJSON<void>("/api/hook-events", { client, event });
}

export function setHookEventEnabled(client: string, event: string, enabled: boolean): Promise<void> {
  return api.patchJSON("/api/hook-events", { client, event, enabled });
}

export function deleteHookEvent(client: string, event: string): Promise<void> {
  return api.del("/api/hook-events", { client, event });
}
