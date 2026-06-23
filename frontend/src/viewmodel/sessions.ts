import { useSessions } from "@/query/sessions";
import { useSessionRoute } from "@/viewmodel/route";
import type { SessionDTO } from "@/api/sessions";

export interface SessionVM extends SessionDTO {
  selected: boolean;
}

export interface SessionsView {
  sessions: SessionVM[];
  isLoading: boolean;
  isError: boolean;
}

// ViewModel: combines server state (query) + route state into view-ready data.
// All derivation lives here, not in components (frontend-mvvm §4).
export function useSessionsView(): SessionsView {
  const { data, isLoading, isError } = useSessions();
  const selectedId = useSessionRoute().sessionId;

  const sessions = (data ?? []).map<SessionVM>((s) => ({
    ...s,
    selected: s.id === selectedId,
  }));

  return { sessions, isLoading, isError };
}

export interface SessionGroup {
  client: string;
  sessions: SessionVM[];
}

// Sessions grouped by client (claude_code, codex_cli, then any others), newest
// first within each group — for the sidebar's collapsible per-client sections.
const CLIENT_ORDER = ["claude_code", "codex_cli"];

export function useSessionGroupsView(): { groups: SessionGroup[]; isLoading: boolean; isError: boolean } {
  const { sessions, isLoading, isError } = useSessionsView();

  const byClient = new Map<string, SessionVM[]>();
  for (const s of sessions) {
    const arr = byClient.get(s.client) ?? [];
    arr.push(s);
    byClient.set(s.client, arr);
  }
  const order = [...CLIENT_ORDER, ...[...byClient.keys()].filter((c) => !CLIENT_ORDER.includes(c))];
  const groups: SessionGroup[] = [];
  for (const client of order) {
    const arr = byClient.get(client);
    if (!arr) continue;
    arr.sort((a, b) => (a.started_at < b.started_at ? 1 : -1));
    groups.push({ client, sessions: arr });
  }
  return { groups, isLoading, isError };
}
