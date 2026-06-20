import { useTranslation } from "react-i18next";
import { useUIStore } from "@/store/ui";
import { useSessionEventsView } from "@/viewmodel/detail";
import { EventTimeline } from "./EventTimeline";
import { EventDetailPanel } from "./EventDetailPanel";
import { HookDetailPanel } from "./HookDetailPanel";
import { DetailLinksProvider } from "./DetailLinks";
import { FlowGraphPanel } from "./FlowGraphPanel";
import { Sheet, SheetContent, SheetA11y } from "@/components/ui/sheet";

// Main pane: the selected session's event timeline + the selected event detail.
// In Flow mode the right pane is the turn's causal flow graph, and node detail
// opens in a side sheet (no fixed detail column) so the graph gets full width.
export function SessionDetail() {
  const { t } = useTranslation();
  const sessionId = useUIStore((s) => s.selectedSessionId);
  const selectedEventId = useUIStore((s) => s.selectedEventId);
  const sheetEventId = useUIStore((s) => s.sheetEventId);
  const timelineMode = useUIStore((s) => s.timelineMode);
  const selectEvent = useUIStore((s) => s.selectEvent);
  const openSheet = useUIStore((s) => s.openSheet);
  const closeSheet = useUIStore((s) => s.closeSheet);
  const { events, defaultEventId, isLoading, isError } = useSessionEventsView(sessionId);

  if (!sessionId) return <Centered>{t("detail.pick_session")}</Centered>;
  if (isLoading) return <Centered>{t("detail.loading")}</Centered>;
  if (isError) return <Centered className="text-error">{t("detail.error")}</Centered>;
  if (events.length === 0) return <Centered>{t("detail.empty")}</Centered>;

  const effectiveId = selectedEventId ?? defaultEventId;
  const selected = events.find((e) => e.id === effectiveId);
  const isHook = selected?.kind === "hook_event";
  const showFlowGraph = timelineMode === "timeline" && events.some((e) => e.kind === "hook_event");
  const sheetEvent = events.find((e) => e.id === sheetEventId);

  return (
    <div className="flex h-full">
      <div className="w-64 shrink-0 overflow-y-auto border-r bg-surface">
        <EventTimeline events={events} selectedId={effectiveId} onSelect={selectEvent} />
      </div>
      <div className="min-w-0 flex-1 overflow-y-auto">
        <DetailLinksProvider events={events}>
          {showFlowGraph ? (
            <FlowGraphPanel events={events} selectedId={effectiveId} onOpenDetail={openSheet} />
          ) : !effectiveId ? (
            <Centered>{t("detail.pick_event")}</Centered>
          ) : isHook ? (
            <HookDetailPanel eventId={effectiveId} />
          ) : (
            <EventDetailPanel eventId={effectiveId} />
          )}

          <Sheet open={!!sheetEventId} onOpenChange={(o: boolean) => !o && closeSheet()}>
            <SheetContent>
              <SheetA11y title={t("detail.request")} />
              <div className="h-full overflow-y-auto">
                {sheetEventId &&
                  (sheetEvent?.kind === "hook_event" ? (
                    <HookDetailPanel eventId={sheetEventId} />
                  ) : (
                    <EventDetailPanel eventId={sheetEventId} />
                  ))}
              </div>
            </SheetContent>
          </Sheet>
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
