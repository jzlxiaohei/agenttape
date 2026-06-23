import { lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { useEventDetailView } from "@/viewmodel/detail";
import { useCompactionEpisodes } from "@/query/events";
import { useUIStore } from "@/store/ui";
import { TokenBars } from "./TokenBars";
import { TagList } from "./TagList";
import { CompactionPanel } from "./CompactionPanel";
import { CopyButton } from "./CopyButton";
import { MessageThread } from "./MessageThread";
import { ContentBlocks } from "./ContentBlocks";
import { ToolsView } from "./ToolsView";
import { Collapsible } from "./Collapsible";
import { RoundsView } from "./RoundsView";
import { DetailFilterBar } from "./DetailFilterBar";
import { DetailOutline, type OutlineItem } from "./DetailOutline";
import { Tabs } from "./Tabs";

const RawView = lazy(() => import("./RawView").then((m) => ({ default: m.RawView })));
const DiffView = lazy(() => import("./DiffView"));
const ReplayView = lazy(() => import("./ReplayView"));

const BIG = 20;

// One event's detail. Event-level header (real usage + tags) sits above the
// Request | Response | Raw tabs, so the request-only composition lives clearly
// inside the Request tab and never gets confused with overall usage.
export function EventDetailPanel({ eventId }: { eventId: string }) {
  const { t } = useTranslation();
  const vm = useEventDetailView(eventId);
  const tab = useUIStore((s) => s.detailTab);
  const setTab = useUIStore((s) => s.setDetailTab);
  // Compaction is a cross-event judgment: this event shows the panel only if it's
  // the "before" (summarize-trigger) of a detected episode.
  const { data: episodes } = useCompactionEpisodes(vm.sessionId || null);
  const episode = (episodes ?? []).find((e) => e.before_event === eventId);

  if (vm.isLoading) return <p className="p-6 text-muted-foreground">{t("detail.loading")}</p>;
  if (vm.isError || !vm.found || !vm.header)
    return <p className="p-6 text-error">{t("detail.error")}</p>;

  const h = vm.header;
  const c = vm.counts;

  return (
    <div className="flex flex-col">
      <header className="space-y-3 p-6 pb-3">
        <div className="flex items-center gap-2">
          <span className="text-lg font-semibold">{h.model || h.provider || t("detail.event")}</span>
          <span className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">{h.provider}</span>
        </div>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
          <span>{t("detail.status")}: {h.status}</span>
          <span>{t("detail.duration")}: {h.durationMs}ms</span>
          {vm.usage && (
            <span className="mono">
              {t("detail.usage")}: in {vm.usage.input_tokens ?? 0} · out {vm.usage.output_tokens ?? 0} ·
              total {vm.usage.total_tokens ?? 0}
            </span>
          )}
        </div>
        {h.target && (
          <div className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
            <span className="mono min-w-0 truncate" title={`${h.method} ${h.target}`}>
              {h.method} {h.target}
            </span>
            <CopyButton text={h.target} />
          </div>
        )}
        <TagList tags={vm.tags} />
      </header>

      {episode && vm.compactionMetrics && (
        <div className="px-6 pb-2">
          <CompactionPanel grade={episode.grade} evidence={episode.evidence} data={vm.compactionMetrics} />
        </div>
      )}

      <div className="px-6">
        <Tabs
          items={[
            { key: "request", label: t("detail.request"), count: c.system + c.tools + c.requestMessages },
            { key: "response", label: t("detail.response"), count: c.responseMessages },
            { key: "raw", label: t("detail.raw") },
            { key: "diff", label: t("detail.diff") },
            { key: "replay", label: t("detail.replay") },
          ]}
          active={tab}
          onChange={(k) => setTab(k as typeof tab)}
        />
      </div>

      {tab === "request" && <RequestTab vm={vm} />}
      {tab === "response" && (
        <div className="p-6">
          <MessageThread messages={vm.groups.responseMessages} />
        </div>
      )}
      {tab === "raw" && (
        <div className="p-6">
          <Suspense fallback={<p className="text-xs text-muted-foreground">{t("raw.loading")}</p>}>
            <RawView eventId={eventId} />
          </Suspense>
        </div>
      )}
      {tab === "diff" && (
        <div className="p-6">
          <Suspense fallback={<p className="text-xs text-muted-foreground">{t("raw.loading")}</p>}>
            <DiffView eventId={eventId} />
          </Suspense>
        </div>
      )}
      {tab === "replay" && (
        <div className="p-6">
          <Suspense fallback={<p className="text-xs text-muted-foreground">{t("raw.loading")}</p>}>
            <ReplayView eventId={eventId} />
          </Suspense>
        </div>
      )}
    </div>
  );
}

function RequestTab({ vm }: { vm: ReturnType<typeof useEventDetailView> }) {
  const { t } = useTranslation();
  const parts = useUIStore((s) => s.parts);
  const groupRounds = useUIStore((s) => s.groupRounds);
  const g = vm.groups;
  const c = vm.counts;

  const outline: OutlineItem[] = [];
  if (parts.system && c.system) outline.push({ key: "system", label: t("section.system"), count: c.system });
  if (parts.tools && c.tools) outline.push({ key: "tools", label: t("section.tools"), count: c.tools });
  if (parts.messages && c.requestMessages) {
    if (groupRounds) {
      for (const r of g.requestRounds) {
        outline.push({
          key: r.key,
          label: r.index === 0 ? t("round.preamble") : t("round.n", { n: r.index }),
          count: r.messages.length,
        });
      }
    } else {
      outline.push({ key: "reqmsgs", label: t("section.messages"), count: c.requestMessages });
    }
  }

  return (
    <div className="flex">
      <div className="min-w-0 flex-1 space-y-2 p-6">
        <div>
          <p className="mb-1 text-xs text-muted-foreground">{t("detail.composition")}</p>
          <TokenBars bars={vm.sectionBars} />
        </div>
        <DetailFilterBar />

        {parts.system && c.system > 0 && (
          <Collapsible sectionKey="system" title={t("section.system")} count={c.system} accent="var(--color-accent)">
            <ContentBlocks blocks={g.system} />
          </Collapsible>
        )}
        {parts.tools && c.tools > 0 && (
          <Collapsible sectionKey="tools" title={t("section.tools")} count={c.tools} accent="var(--color-toolcall)" defaultCollapsed>
            <ToolsView tools={g.tools} />
          </Collapsible>
        )}
        {parts.messages && c.requestMessages > 0 && groupRounds && (
          <div className="space-y-1">
            <p className="pt-2 text-sm font-semibold">
              {t("section.messages")} <span className="text-xs text-muted-foreground">{c.requestMessages}</span>
            </p>
            <RoundsView rounds={g.requestRounds} />
          </div>
        )}
        {parts.messages && c.requestMessages > 0 && !groupRounds && (
          <Collapsible
            sectionKey="reqmsgs"
            title={t("section.messages")}
            count={c.requestMessages}
            defaultCollapsed={c.requestMessages > BIG}
          >
            <MessageThread messages={g.requestMessages} />
          </Collapsible>
        )}
      </div>
      <aside className="hidden w-44 shrink-0 border-l px-2 lg:block">
        <DetailOutline items={outline} />
      </aside>
    </div>
  );
}
