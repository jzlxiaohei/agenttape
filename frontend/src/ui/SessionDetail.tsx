import { useTranslation } from "react-i18next";
import { useUIStore } from "@/store/ui";
import { useSessionEventsView } from "@/viewmodel/detail";
import { EventTimeline } from "./EventTimeline";
import { EventDetailPanel } from "./EventDetailPanel";
import { HookDetailPanel } from "./HookDetailPanel";
import { DetailLinksProvider } from "./DetailLinks";

// Main pane: the selected session's event timeline + the selected event detail.
// Selection defaults to the latest completion (computed in the viewmodel), with
// no local effect/state — the effective id is derived.
export function SessionDetail() {
  const { t } = useTranslation();
  const sessionId = useUIStore((s) => s.selectedSessionId);
  const selectedEventId = useUIStore((s) => s.selectedEventId);
  const selectEvent = useUIStore((s) => s.selectEvent);
  const { events, defaultEventId, isLoading, isError } = useSessionEventsView(sessionId);

  if (!sessionId)
    return <Centered>{t("detail.pick_session")}</Centered>;
  if (isLoading) return <Centered>{t("detail.loading")}</Centered>;
  if (isError) return <Centered className="text-error">{t("detail.error")}</Centered>;
  if (events.length === 0) return <Centered>{t("detail.empty")}</Centered>;

  const effectiveId = selectedEventId ?? defaultEventId;
  const selected = events.find((e) => e.id === effectiveId);
  const isHook = selected?.kind === "hook_event";

  return (
    <div className="flex h-full">
      <div className="w-64 shrink-0 overflow-y-auto border-r bg-surface">
        <EventTimeline events={events} selectedId={effectiveId} onSelect={selectEvent} />
      </div>
      <div className="min-w-0 flex-1 overflow-y-auto">
        <DetailLinksProvider events={events}>
          {!effectiveId ? (
            <Centered>{t("detail.pick_event")}</Centered>
          ) : isHook ? (
            <HookDetailPanel eventId={effectiveId} />
          ) : (
            <EventDetailPanel eventId={effectiveId} />
          )}
        </DetailLinksProvider>
      </div>
    </div>
  );
}

function Centered({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={`flex h-full items-center justify-center p-6 text-muted-foreground ${className ?? ""}`}>
      {children}
    </div>
  );
}
