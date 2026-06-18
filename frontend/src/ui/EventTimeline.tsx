import { useTranslation } from "react-i18next";
import type { EventSummary } from "@/api/events";
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
  return (
    <ul className="divide-y">
      {events.map((e) => (
        <li
          key={e.id}
          onClick={() => onSelect(e.id)}
          className={cn(
            "cursor-pointer px-3 py-2 transition-colors hover:bg-muted/60",
            selectedId === e.id && "bg-accent/8",
            !e.is_completion && "opacity-60",
          )}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium">
              {e.is_completion ? e.model || e.provider : t("event.probe")}
            </span>
            <span className="shrink-0 text-xs text-muted-foreground mono">{formatTime(e.started_at)}</span>
          </div>
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-xs text-muted-foreground mono">
              {e.method} {endpointOf(e.target)}
            </span>
            {e.total_tokens > 0 && (
              <span className="shrink-0 text-xs text-muted-foreground mono">{e.total_tokens}</span>
            )}
          </div>
        </li>
      ))}
    </ul>
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
