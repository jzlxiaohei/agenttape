import { useTranslation } from "react-i18next";
import type { EventSummary } from "@/api/events";
import { groupIntoTurns } from "@/viewmodel/detail";
import { useUIStore } from "@/store/ui";
import { cn } from "@/lib/utils";

// Left rail for the selected session. Requests mode is a model-call browser;
// Flow mode only selects a turn because the right pane renders the graph.
export function EventTimeline({
  events,
  selectedId,
  onSelect,
}: {
  events: EventSummary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const { t } = useTranslation();
  const mode = useUIStore((s) => s.timelineMode);
  const setMode = useUIStore((s) => s.setTimelineMode);

  const hasHooks = events.some((e) => e.kind === "hook_event");
  const isFlowMode = hasHooks && mode === "timeline";
  const turns = groupIntoTurns(events);
  const activeHint = isFlowMode ? t("timeline.timeline_hint") : t("timeline.requests_hint");

  const header = hasHooks ? (
    <div className="border-b p-2">
      <div className="flex gap-1 text-xs">
        {(["requests", "timeline"] as const).map((m) => (
          <button
            key={m}
            onClick={() => setMode(m)}
            className={cn(
              "flex-1 rounded-md border px-2 py-1",
              mode === m ? "border-accent bg-accent/10 text-accent" : "text-muted-foreground hover:bg-muted",
            )}
          >
            {t(`timeline.${m}`)}
          </button>
        ))}
      </div>
      <p className="mt-1.5 px-1 text-[11px] leading-snug text-muted-foreground">{activeHint}</p>
    </div>
  ) : null;

  if (isFlowMode && turns) {
    return (
      <div>
        {header}
        {[...turns].reverse().map((turn, i) => {
          const isLatest = i === 0;
          const hasSelected = turn.events.some((e) => e.id === selectedId);
          const title = turn.index === 0 ? t("turn.session") : t("turn.n", { n: turn.index });
          return (
            <button
              key={turn.key}
              onClick={() => turn.events[0] && onSelect(turn.events[0].id)}
              className={cn(
                "block w-full border-b px-3 py-2 text-left transition-colors hover:bg-muted/60",
                (hasSelected || (!selectedId && isLatest)) && "bg-accent/8",
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

  const list = isFlowMode
    ? [...events].sort((a, b) => a.started_at.localeCompare(b.started_at))
    : events.filter((e) => e.kind !== "hook_event");

  return (
    <div>
      {header}
      <ul className="divide-y">
        {list.map((e) => (
          <EventRow key={e.id} ev={e} selected={selectedId === e.id} onSelect={() => onSelect(e.id)} />
        ))}
      </ul>
    </div>
  );
}

function EventRow({ ev, selected, onSelect }: { ev: EventSummary; selected: boolean; onSelect: () => void }) {
  const { t } = useTranslation();
  const isHook = ev.kind === "hook_event";
  return (
    <li
      onClick={onSelect}
      className={cn(
        "cursor-pointer px-3 py-2 transition-colors hover:bg-muted/60",
        selected && "bg-accent/8",
        !isHook && !ev.is_completion && "opacity-60",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className={cn("truncate text-sm font-medium", isHook && "text-accent/90")}>
          {isHook ? ev.hook_event : ev.is_completion ? ev.model || ev.provider : t("event.probe")}
        </span>
        <span className="shrink-0 text-xs text-muted-foreground mono">{formatTime(ev.started_at)}</span>
      </div>
      {!isHook && (
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-xs text-muted-foreground mono">
            {ev.method} {endpointOf(ev.target)}
          </span>
          {ev.total_tokens > 0 && (
            <span className="shrink-0 text-xs text-muted-foreground mono">{ev.total_tokens}</span>
          )}
        </div>
      )}
      {isHook && ev.tool_call_id && (
        <div className="truncate text-xs text-muted-foreground mono">{ev.tool_call_id}</div>
      )}
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
