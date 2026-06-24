import { useSessions } from "@/query/sessions";
import { useSessionRoute } from "@/viewmodel/route";
import { useUIStore } from "@/store/ui";
import type { SessionDTO } from "@/api/sessions";

// matchesSessionFilter prioritizes the name (title): a session matches if its title
// or its id contains the (lowercased) query. Empty query matches everything.
function matchesSessionFilter(s: SessionDTO, q: string): boolean {
  if (!q) return true;
  const needle = q.toLowerCase();
  return s.title.toLowerCase().includes(needle) || s.id.toLowerCase().includes(needle);
}

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
  const filter = useUIStore((s) => s.sessionFilter);

  const sessions = (data ?? [])
    .filter((s) => matchesSessionFilter(s, filter))
    .map<SessionVM>((s) => ({
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
