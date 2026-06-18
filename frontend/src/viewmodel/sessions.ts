import { useSessions } from "@/query/sessions";
import { useUIStore } from "@/store/ui";
import type { SessionDTO } from "@/api/sessions";

export interface SessionVM extends SessionDTO {
  selected: boolean;
}

export interface SessionsView {
  sessions: SessionVM[];
  isLoading: boolean;
  isError: boolean;
}

// ViewModel: combines server state (query) + UI state (store) into view-ready
// data. All derivation lives here, not in components (frontend-mvvm §4).
export function useSessionsView(): SessionsView {
  const { data, isLoading, isError } = useSessions();
  const selectedId = useUIStore((s) => s.selectedSessionId);

  const sessions = (data ?? []).map<SessionVM>((s) => ({
    ...s,
    selected: s.id === selectedId,
  }));

  return { sessions, isLoading, isError };
}
