import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Webhook, Cpu, ChevronRight, ChevronDown } from "lucide-react";
import type { EventSummary } from "@/api/events";
import { groupIntoTurns, buildTurnFlow, type Turn, type FlowHookNode } from "@/viewmodel/detail";
import { useRawFile } from "@/query/events";

// Hook-first flow: the harness hook timeline is the spine (orchestration), each
// node showing its full payload inline. http is not a first-layer node — a node
// that relates to a request shows a "request #N" chip that opens the http detail
// in the side sheet. Several hooks can point to the same request.
export function FlowGraphPanel({
  events,
  selectedId,
  onOpenDetail,
}: {
  events: EventSummary[];
  selectedId: string | null;
  onOpenDetail: (id: string) => void;
}) {
  const { t } = useTranslation();
  const turns = flowTurns(events);
  const active = activeTurn(turns, selectedId);
  if (!active) return <div className="p-6 text-muted-foreground">{t("flow.empty")}</div>;

  const flow = buildTurnFlow(active.events);
  const title = active.index === 0 ? t("turn.session") : t("turn.n", { n: active.index });

  return (
    <div className="h-full overflow-y-auto">
      <header className="border-b px-6 py-4">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <h2 className="text-lg font-semibold">{title}</h2>
          <span className="text-xs text-muted-foreground">
            {t("turn.summary", { http: active.httpCount, hooks: active.hookCount })}
          </span>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">{t("flow.hint")}</p>
      </header>

      <div className="space-y-2 p-6">
        {flow.nodes.length === 0 ? (
          <div className="text-muted-foreground">{t("flow.empty")}</div>
        ) : (
          flow.nodes.map((node) => (
            <HookFlowCard key={node.event.id} node={node} onOpenHttp={onOpenDetail} />
          ))
        )}
      </div>
    </div>
  );
}

function HookFlowCard({ node, onOpenHttp }: { node: FlowHookNode; onOpenHttp: (id: string) => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const ev = node.event;
  const raw = useRawFile(ev.id, "hook_payload", true);
  const fields = extractFields(raw.data);
  const pretty = prettyJSON(raw.data);

  return (
    <div className="rounded-lg border bg-card px-3 py-2.5 shadow-sm">
      <div className="flex items-center gap-2">
        <Webhook size={15} className="shrink-0 text-accent" />
        <span className="text-sm font-semibold text-accent/90">{ev.hook_event}</span>
        {ev.tool_name && (
          <span className="rounded bg-toolcall/10 px-1.5 py-0.5 text-[11px] font-medium text-toolcall mono">
            {ev.tool_name}
          </span>
        )}
        {node.httpRef && (
          <button
            onClick={() => onOpenHttp(node.httpRef!.id)}
            className="inline-flex items-center gap-1 rounded-md border border-toolcall/40 bg-toolcall/5 px-2 py-0.5 text-xs text-toolcall hover:bg-toolcall/10"
          >
            <Cpu size={12} />
            {t("flow.view_request", { n: node.httpRef.index })}
          </button>
        )}
        <span className="ml-auto shrink-0 text-xs text-muted-foreground mono">{formatTime(ev.started_at)}</span>
      </div>

      {ev.tool_call_id && (
        <div className="mt-1 text-[11px] text-muted-foreground">
          <span className="mono">{ev.tool_call_id}</span>
        </div>
      )}

      {raw.isLoading && <p className="mt-2 text-[11px] text-muted-foreground">{t("raw.loading")}</p>}

      {fields.length > 0 && (
        <dl className="mt-2 grid grid-cols-[max-content_1fr] gap-x-3 gap-y-0.5 text-xs">
          {fields.map(([k, v]) => (
            <div key={k} className="contents">
              <dt className="mono text-muted-foreground">{k}</dt>
              <dd className="mono break-words">{v}</dd>
            </div>
          ))}
        </dl>
      )}

      {pretty && (
        <div className="mt-2">
          <button
            onClick={() => setOpen((v) => !v)}
            className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
          >
            {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
            {t("hook.payload")}
          </button>
          {open && (
            <pre className="mt-1 max-h-72 overflow-auto rounded-md bg-muted px-2.5 py-2 text-[11px] leading-relaxed mono">
              {pretty}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

// extractFields pulls the most useful fields out of a hook payload for an
// always-visible summary above the (folded) full payload. It digs into the
// nested tool_input (where Codex/cc put the actual command/args) and previews
// tool_response, so a tool hook shows what it did without expanding.
function extractFields(payload?: string): [string, string][] {
  if (!payload) return [];
  let obj: Record<string, unknown>;
  try {
    obj = JSON.parse(payload);
  } catch {
    return [];
  }
  const out: [string, string][] = [];
  const push = (k: string, v: unknown) => {
    if (v == null || v === "") return;
    out.push([k, truncate(typeof v === "string" ? v : JSON.stringify(v), 240)]);
  };

  push("prompt", obj.prompt);
  const ti = obj.tool_input;
  if (ti && typeof ti === "object") {
    const cmd = (ti as Record<string, unknown>).command;
    if (cmd != null) push("command", cmd);
    else push("input", ti);
  } else if (ti != null) {
    push("input", ti);
  }
  push("response", obj.tool_response);
  push("trigger", obj.trigger);
  push("permission_mode", obj.permission_mode);
  return out;
}

function prettyJSON(payload?: string): string {
  if (!payload) return "";
  try {
    return JSON.stringify(JSON.parse(payload), null, 2);
  } catch {
    return payload;
  }
}

function truncate(s: string, n: number): string {
  return s.length <= n ? s : `${s.slice(0, n)}…`;
}

function flowTurns(events: EventSummary[]): Turn[] {
  const turns = groupIntoTurns(events);
  if (turns) return turns;
  const ordered = [...events].sort((a, b) => a.started_at.localeCompare(b.started_at));
  if (ordered.length === 0) return [];
  return [
    {
      key: "flow-all",
      index: 1,
      events: ordered,
      httpCount: ordered.filter((e) => e.kind !== "hook_event").length,
      hookCount: ordered.filter((e) => e.kind === "hook_event").length,
      startedAt: ordered[0].started_at,
    },
  ];
}

function activeTurn(turns: Turn[], selectedId: string | null): Turn | null {
  if (selectedId) {
    const selected = turns.find((t) => t.events.some((e) => e.id === selectedId));
    if (selected) return selected;
  }
  return turns[turns.length - 1] ?? null;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}
