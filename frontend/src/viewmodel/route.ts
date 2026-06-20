import { useNavigate, useParams, useSearchParams } from "react-router-dom";

// Navigation state lives in the URL (source of truth), so paths are shareable and
// survive reload / back-forward:
//   /sessions                          list only
//   /sessions/:id                      a session (requests tab, latest request)
//   /sessions/:id?tab=requests&req_id  a focused request (inline detail)
//   /sessions/:id?tab=flow&turn_id     a turn's flow graph
//   /sessions/:id?tab=flow&turn_id&req_id   …with a request open in the side sheet
// Non-navigation UI state (filters, folds, markdown) stays in the Zustand store.
export type FlowTab = "requests" | "flow";

export interface SessionRoute {
  sessionId: string | null;
  tab: FlowTab;
  reqId: string | null;
  turnId: string | null;
  openSession: (id: string) => void;
  setTab: (tab: FlowTab) => void;
  selectRequest: (id: string) => void; // requests tab: inline focus · flow tab: opens the sheet
  selectTurn: (id: string) => void;
  closeSheet: () => void; // clears req_id (the flow side sheet)
  openEvent: (sessionId: string, eventId: string) => void; // cross-session jump (search)
}

export function useSessionRoute(): SessionRoute {
  const { id } = useParams();
  const [sp, setSp] = useSearchParams();
  const navigate = useNavigate();
  const tab: FlowTab = sp.get("tab") === "flow" ? "flow" : "requests";

  const patch = (mut: (p: URLSearchParams) => void) => {
    const p = new URLSearchParams(sp);
    mut(p);
    setSp(p);
  };

  return {
    sessionId: id ?? null,
    tab,
    reqId: sp.get("req_id"),
    turnId: sp.get("turn_id"),
    openSession: (sid) => navigate(`/sessions/${sid}`),
    setTab: (t) => patch((p) => p.set("tab", t)),
    selectRequest: (rid) => patch((p) => p.set("req_id", rid)),
    selectTurn: (tid) => patch((p) => p.set("turn_id", tid)),
    closeSheet: () => patch((p) => p.delete("req_id")),
    openEvent: (sid, eid) => navigate(`/sessions/${sid}?tab=requests&req_id=${eid}`),
  };
}
