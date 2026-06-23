import { api } from "./client";

// Codex desktop routing: unlike cc/codex CLI (env / -c overrides), the desktop app
// can only be routed by writing ~/.codex/config.toml — so the server backs it up,
// injects, and restores. These endpoints drive that lifecycle.
export interface CodexDesktopStatus {
  active: boolean;
  session_id?: string;
  config_path?: string;
  installed_at?: string;
  hooks?: boolean;
}

export interface CodexDesktopInstallResult {
  ok: boolean;
  session_id: string;
  config_path: string;
  backup_path: string;
  had_original: boolean;
  hooks: boolean;
}

export function fetchCodexDesktopStatus(): Promise<CodexDesktopStatus> {
  return api.getJSON<CodexDesktopStatus>("/api/codex-desktop/status");
}

export function installCodexDesktop(hooks: boolean): Promise<CodexDesktopInstallResult> {
  return api.postJSON<CodexDesktopInstallResult>("/api/codex-desktop/install", { hooks });
}

export function restoreCodexDesktop(): Promise<{ ok: boolean; restored: boolean }> {
  return api.postJSON<{ ok: boolean; restored: boolean }>("/api/codex-desktop/restore", {});
}
