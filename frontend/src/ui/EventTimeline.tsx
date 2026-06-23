import { useTranslation } from "react-i18next";
import { Globe } from "lucide-react";
import type { EventSummary } from "@/api/events";
import { groupIntoTurns } from "@/viewmodel/detail";
import { useSessionRoute, type FlowTab } from "@/viewmodel/route";
import { cn } from "@/lib/utils";

// Left rail for the selected session. requests tab is a model-call browser; flow
// tab only selects a turn (the right pane renders the graph). Selection + tab live
// in the URL (route.ts).
export function EventTimeline({ events }: { events: EventSummary[] }) {
  const { t } = useTranslation();
  const route = useSessionRoute();

  const hasHooks = events.some((e) => e.kind === "hook_event");
  const isFlow = hasHooks && route.tab === "flow";
  const turns = groupIntoTurns(events);
  const activeHint = isFlow ? t("timeline.flow_hint") : t("timeline.requests_hint");
  const latestCompletionId = events.find((e) => e.is_completion)?.id ?? null;
  const reqSelected = route.reqId ?? latestCompletionId;

  const header = hasHooks ? (
    <div className="border-b p-2">
      <div className="flex gap-1 text-xs">
        {(["requests", "flow"] as const).map((m: FlowTab) => (
          <button
            key={m}
            onClick={() => route.setTab(m)}
            className={cn(
              "flex-1 rounded-md border px-2 py-1",
              route.tab === m ? "border-accent bg-accent/10 text-accent" : "text-muted-foreground hover:bg-muted",
            )}
          >
            {t(`timeline.${m}`)}
          </button>
        ))}
      </div>
      <p className="mt-1.5 px-1 text-[11px] leading-snug text-muted-foreground">{activeHint}</p>
    </div>
  ) : (
    // No hooks → HTTP-only capture (env "run yourself"). Flag it so the missing flow
    // tab is explained rather than just absent.
    <div className="flex items-center gap-1.5 border-b px-3 py-2" title={t("sessions.http_only_hint")}>
      <Globe size={13} className="shrink-0 text-muted-foreground" />
      <span className="text-xs font-medium text-muted-foreground">{t("sessions.http_only")}</span>
    </div>
  );

  if (isFlow && turns) {
    return (
      <div>
        {header}
        {[...turns].reverse().map((turn, i) => {
          const isLatest = i === 0;
          const hasSelected = turn.events.some((e) => e.id === route.turnId);
          const title = turn.index === 0 ? t("turn.session") : t("turn.n", { n: turn.index });
          return (
            <button
              key={turn.key}
              onClick={() => turn.events[0] && route.selectTurn(turn.events[0].id)}
              className={cn(
                "block w-full border-b px-3 py-2 text-left transition-colors hover:bg-muted/60",
                (hasSelected || (!route.turnId && isLatest)) && "bg-accent/8",
              )}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="truncate text-sm font-semibold">{title}</span>
                <span className="shrink-0 text-xs text-muted-foreground mono">{formatTime(turn.startedAt)}</span>
              </div>
              <div className="mt-0.5 text-xs text-muted-foreground">
                {t("turn.summary", { http: turn.httpCount, hooks: turn.hookCount })}
              </div>
            </button>
          );
        })}
      </div>
    );
  }

  const list = events.filter((e) => e.kind !== "hook_event");
  return (
    <div>
      {header}
      <ul className="divide-y">
        {list.map((e) => (
          <EventRow key={e.id} ev={e} selected={reqSelected === e.id} onSelect={() => route.selectRequest(e.id)} />
        ))}
      </ul>
    </div>
  );
}

function EventRow({ ev, selected, onSelect }: { ev: EventSummary; selected: boolean; onSelect: () => void }) {
  const { t } = useTranslation();
  return (
    <li
      onClick={onSelect}
      className={cn(
        "cursor-pointer px-3 py-2 transition-colors hover:bg-muted/60",
        selected && "bg-accent/8",
        !ev.is_completion && "opacity-60",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-sm font-medium">
          {ev.is_completion ? ev.model || ev.provider : t("event.probe")}
        </span>
        <span className="shrink-0 text-xs text-muted-foreground mono">{formatTime(ev.started_at)}</span>
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-xs text-muted-foreground mono">
          {ev.method} {endpointOf(ev.target)}
        </span>
        {ev.total_tokens > 0 && (
          <span className="shrink-0 text-xs text-muted-foreground mono">{ev.total_tokens}</span>
        )}
      </div>
    </li>
  );
}

function endpointOf(target: string): string {
  try {
    return new URL(target).pathname;
  } catch {
    return target;
  }
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}
