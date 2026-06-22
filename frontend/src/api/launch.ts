import { api } from "./client";

export type LaunchKind = "cc" | "codex";
export type LaunchMode = "subscription" | "key";

export interface LaunchReq {
  kind: LaunchKind;
  workdir?: string;
  mode?: LaunchMode;
  upstream?: string;
  api_key?: string;
  terminal?: string;
}

// fetchTerminals lists installed terminal apps to launch into (macOS).
export function fetchTerminals(): Promise<string[]> {
  return api.getJSON<string[]>("/api/terminals").then((t) => t ?? []);
}

export interface LaunchPreview {
  command: string;
  enabled: boolean; // whether server-side launch is allowed (-allow-launch)
}

// previewLaunch returns the copy-paste command without running anything.
export function previewLaunch(req: LaunchReq): Promise<LaunchPreview> {
  return api.postJSON<LaunchPreview>("/api/launch", { ...req, preview: true });
}

// launchAgent asks the server to start a coding agent (new terminal), routed
// through the proxy. The session self-registers and shows up in Sessions shortly.
export function launchAgent(req: LaunchReq): Promise<{ ok: boolean }> {
  return api.postJSON<{ ok: boolean }>("/api/launch", req);
}
