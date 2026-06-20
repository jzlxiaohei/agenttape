import { useTranslation } from "react-i18next";
import { useSessionRoute } from "@/viewmodel/route";
import { useSessionEventsView } from "@/viewmodel/detail";
import { EventTimeline } from "./EventTimeline";
import { EventDetailPanel } from "./EventDetailPanel";
import { HookDetailPanel } from "./HookDetailPanel";
import { DetailLinksProvider } from "./DetailLinks";
import { FlowGraphPanel } from "./FlowGraphPanel";
import { Sheet, SheetContent, SheetA11y } from "@/components/ui/sheet";

// Main pane: a session's timeline + detail, all driven by the URL (see route.ts).
// requests tab → inline detail of req_id; flow tab → the turn graph, with req_id
// (if any) opening the http exchange in a side sheet.
export function SessionDetail() {
  const { t } = useTranslation();
  const route = useSessionRoute();
  const { events, defaultEventId, isLoading, isError } = useSessionEventsView(route.sessionId);

  if (!route.sessionId) return <Centered>{t("detail.pick_session")}</Centered>;
  if (isLoading) return <Centered>{t("detail.loading")}</Centered>;
  if (isError) return <Centered className="text-error">{t("detail.error")}</Centered>;
  if (events.length === 0) return <Centered>{t("detail.empty")}</Centered>;

  const hasHooks = events.some((e) => e.kind === "hook_event");
  const showFlow = route.tab === "flow" && hasHooks;

  // requests tab: the inline-focused event (falls back to latest completion)
  const reqEventId = route.reqId ?? defaultEventId;
  const reqEvent = events.find((e) => e.id === reqEventId);

  // flow tab: req_id (if present) is the request open in the side sheet
  const sheetOpen = showFlow && !!route.reqId;
  const sheetEvent = events.find((e) => e.id === route.reqId);

  return (
    <div className="flex h-full">
      <div className="w-64 shrink-0 overflow-y-auto border-r bg-surface">
        <EventTimeline events={events} />
      </div>
      <div className="min-w-0 flex-1 overflow-y-auto">
        <DetailLinksProvider events={events}>
          {showFlow ? (
            <FlowGraphPanel events={events} selectedId={route.turnId} onOpenDetail={route.selectRequest} />
          ) : !reqEventId ? (
            <Centered>{t("detail.pick_event")}</Centered>
          ) : reqEvent?.kind === "hook_event" ? (
            <HookDetailPanel eventId={reqEventId} />
          ) : (
            <EventDetailPanel eventId={reqEventId} />
          )}

          <Sheet open={sheetOpen} onOpenChange={(o: boolean) => !o && route.closeSheet()}>
            <SheetContent>
              <SheetA11y title={t("detail.request")} />
              <div className="h-full overflow-y-auto">
                {route.reqId &&
                  (sheetEvent?.kind === "hook_event" ? (
                    <HookDetailPanel eventId={route.reqId} />
                  ) : (
                    <EventDetailPanel eventId={route.reqId} />
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
