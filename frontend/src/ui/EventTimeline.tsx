import { useTranslation } from "react-i18next";
import { Webhook } from "lucide-react";
import type { EventSummary } from "@/api/events";
import { groupIntoTurns } from "@/viewmodel/detail";
import { useUIStore } from "@/store/ui";
import { Collapsible } from "./Collapsible";
import { cn } from "@/lib/utils";

// The selected session's events as a chronological list. Control/probe requests
// are visually de-emphasized vs real completions (next.md: don't mix them).
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
  const turns = groupIntoTurns(events);
  const showTimeline = hasHooks && mode === "timeline" && turns;

  // Default (and http-only sessions): a clean Requests list — hooks stay out of
  // the primary stream. The orchestration view is one toggle away.
  const header = hasHooks ? (
    <div className="flex gap-1 border-b p-2 text-xs">
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
  ) : null;

  if (!showTimeline) {
    const list = mode === "requests" ? events.filter((e) => e.kind !== "hook_event") : events;
    return (
      <div>
        {header}
        <ul className="divide-y">{list.map((e) => row(e, selectedId, onSelect))}</ul>
      </div>
    );
  }

  // Timeline view: group by turn, newest turn on top; events within a turn read
  // chronologically.
  return (
    <div>
      {header}
      {[...turns].reverse().map((turn, i) => {
        const isLatest = i === 0;
        const hasSelected = turn.events.some((e) => e.id === selectedId);
        const title =
          turn.index === 0 ? t("turn.session") : t("turn.n", { n: turn.index });
        return (
          <div key={turn.key} className="border-b px-2">
            <Collapsible
              sectionKey={turn.key}
              title={title}
              subtitle={t("turn.summary", { http: turn.httpCount, hooks: turn.hookCount })}
              defaultCollapsed={!isLatest && !hasSelected}
            >
              <ul className="-mx-2 divide-y border-t">
                {turn.events.map((e) => row(e, selectedId, onSelect))}
              </ul>
            </Collapsible>
          </div>
        );
      })}
    </div>
  );
}

function row(e: EventSummary, selectedId: string | null, onSelect: (id: string) => void) {
  return e.kind === "hook_event" ? (
    <HookRow key={e.id} ev={e} selected={selectedId === e.id} onSelect={() => onSelect(e.id)} />
  ) : (
    <HTTPRow key={e.id} ev={e} selected={selectedId === e.id} onSelect={() => onSelect(e.id)} />
  );
}

function HTTPRow({ ev, selected, onSelect }: { ev: EventSummary; selected: boolean; onSelect: () => void }) {
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

// Hook rows are visually distinct (webhook icon, accent tint) from HTTP rows.
function HookRow({ ev, selected, onSelect }: { ev: EventSummary; selected: boolean; onSelect: () => void }) {
  return (
    <li
      onClick={onSelect}
      className={cn(
        "flex cursor-pointer items-center gap-2 px-3 py-1.5 transition-colors hover:bg-muted/60",
        selected && "bg-accent/8",
      )}
    >
      <Webhook size={13} className="shrink-0 text-accent/70" />
      <span className="truncate text-xs font-medium text-accent/90">{ev.hook_event}</span>
      <span className="ml-auto shrink-0 text-xs text-muted-foreground mono">{formatTime(ev.started_at)}</span>
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
