import { createContext, useContext, useMemo } from "react";
import type { EventSummary } from "@/api/events";

// Cross-links between the two layers of a session: a tool_call in an HTTP
// request and the harness hooks (PreToolUse/PostToolUse) for that same tool, via
// tool_call_id. Provided once at the session level so deep block renderers can
// look up links without prop-drilling.
interface DetailLinks {
  hookForToolCall: (toolCallId?: string) => EventSummary | null;
  requestBeforeHook: (hookEventId: string) => EventSummary | null;
}

const noop: DetailLinks = { hookForToolCall: () => null, requestBeforeHook: () => null };
const Ctx = createContext<DetailLinks>(noop);

export function useDetailLinks(): DetailLinks {
  return useContext(Ctx);
}

export function DetailLinksProvider({
  events,
  children,
}: {
  events: EventSummary[];
  children: React.ReactNode;
}) {
  const value = useMemo<DetailLinks>(() => {
    const hooks = events.filter((e) => e.kind === "hook_event" && e.tool_call_id);
    const byId = (id?: string) => {
      if (!id) return null;
      const matches = hooks.filter((h) => h.tool_call_id === id);
      return matches.find((h) => h.hook_event === "PreToolUse") ?? matches[0] ?? null;
    };
    const completions = events
      .filter((e) => e.is_completion)
      .sort((a, b) => a.started_at.localeCompare(b.started_at));
    const requestBeforeHook = (hookEventId: string) => {
      const hook = events.find((e) => e.id === hookEventId);
      if (!hook) return null;
      let prev: EventSummary | null = null;
      for (const c of completions) {
        if (c.started_at <= hook.started_at) prev = c;
        else break;
      }
      return prev;
    };
    return { hookForToolCall: byId, requestBeforeHook };
  }, [events]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}
