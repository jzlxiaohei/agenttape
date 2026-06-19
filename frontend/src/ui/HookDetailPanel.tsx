import { lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { Webhook, ArrowUpRight } from "lucide-react";
import { useEventDetail, useRawFile } from "@/query/events";
import { useUIStore } from "@/store/ui";
import { useDetailLinks } from "./DetailLinks";
import { TagList } from "./TagList";

const CodeViewer = lazy(() => import("./CodeViewer"));

// Hook events are not LLM exchanges, so they get their own view: the harness
// event name, key payload fields pulled out, and the full payload. Studying
// these is the orchestration-layer angle (next.md 7.1).
export function HookDetailPanel({ eventId }: { eventId: string }) {
  const { t } = useTranslation();
  const { data, isLoading } = useEventDetail(eventId);
  const raw = useRawFile(eventId, "hook_payload", true);
  const links = useDetailLinks();
  const selectEvent = useUIStore((s) => s.selectEvent);
  const request = links.requestBeforeHook(eventId);

  if (isLoading || !data) return <p className="p-6 text-muted-foreground">{t("detail.loading")}</p>;

  const fields = extractFields(raw.data);
  return (
    <div className="mx-auto max-w-3xl space-y-5 p-6">
      <header className="space-y-2">
        <div className="flex items-center gap-2">
          <Webhook size={18} className="text-accent" />
          <span className="text-lg font-semibold">{data.event_name || t("hook.event")}</span>
          <span className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">
            {t("hook.label")} · {data.runtime}
          </span>
        </div>
        {data.tool_call_id && (
          <p className="text-xs text-muted-foreground">
            {t("hook.tool_call_id")}: <span className="mono">{data.tool_call_id}</span>
          </p>
        )}
        {request && (
          <button
            onClick={() => selectEvent(request.id)}
            className="inline-flex items-center gap-1 rounded-md border border-accent/40 px-2 py-0.5 text-xs text-accent hover:bg-accent/10"
          >
            <ArrowUpRight size={12} />
            {t("link.to_request")}
          </button>
        )}
        <TagList tags={data.tags} />
      </header>

      {fields.length > 0 && (
        <section className="space-y-1">
          <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t("hook.fields")}
          </h2>
          <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-sm">
            {fields.map(([k, v]) => (
              <div key={k} className="contents">
                <dt className="mono text-muted-foreground">{k}</dt>
                <dd className="mono break-words">{v}</dd>
              </div>
            ))}
          </dl>
        </section>
      )}

      <section className="space-y-1">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t("hook.payload")}
        </h2>
        {raw.isLoading ? (
          <p className="text-xs text-muted-foreground">{t("raw.loading")}</p>
        ) : (
          <Suspense fallback={<p className="text-xs text-muted-foreground">{t("raw.loading")}</p>}>
            <CodeViewer text={raw.data ?? ""} filename={`${eventId}.hook.json`} height="320px" />
          </Suspense>
        )}
      </section>
    </div>
  );
}

// extractFields pulls a few human-useful fields out of the hook payload for an
// at-a-glance summary (best-effort; full payload is always shown below).
function extractFields(payload?: string): [string, string][] {
  if (!payload) return [];
  let obj: Record<string, unknown>;
  try {
    obj = JSON.parse(payload);
  } catch {
    return [];
  }
  const keys = ["hook_event_name", "tool_name", "permission_mode", "cwd"];
  const out: [string, string][] = [];
  for (const k of keys) {
    if (obj[k] != null) out.push([k, String(obj[k])]);
  }
  return out;
}
