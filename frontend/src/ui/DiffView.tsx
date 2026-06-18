import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { History, CircleDot } from "lucide-react";
import { MergeView } from "@codemirror/merge";
import { EditorView, lineNumbers } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { json } from "@codemirror/lang-json";
import { useUIStore } from "@/store/ui";
import { useSessionEventsView } from "@/viewmodel/detail";
import { useRawFile } from "@/query/events";
import { SemanticDiff } from "./SemanticDiff";
import { cn } from "@/lib/utils";

// Default export so it can be lazy-loaded (CodeMirror merge is heavy). Compares
// the current completion's request input against an earlier completion's, so the
// reader sees exactly what the harness changed turn-to-turn (next.md 5.1).
export default function DiffView({ eventId }: { eventId: string }) {
  const { t } = useTranslation();
  const sessionId = useUIStore((s) => s.selectedSessionId);
  const { events } = useSessionEventsView(sessionId);

  // events are newest-first; completions only.
  const completions = events.filter((e) => e.is_completion);
  const curIdx = completions.findIndex((e) => e.id === eventId);
  const olderCompletions = curIdx >= 0 ? completions.slice(curIdx + 1) : [];

  const [leftId, setLeftId] = useState<string | null>(null);
  const [mode, setMode] = useState<"text" | "semantic">("text");
  const effectiveLeft = leftId ?? olderCompletions[0]?.id ?? null;

  const left = useRawFile(effectiveLeft, "request_body", effectiveLeft !== null && mode === "text");
  const right = useRawFile(eventId, "request_body", mode === "text");

  if (olderCompletions.length === 0)
    return <p className="text-sm text-muted-foreground">{t("diff.no_prev")}</p>;

  const current = completions[curIdx];

  return (
    <div className="space-y-2">
      <div className="flex gap-1 text-xs">
        {(["text", "semantic"] as const).map((m) => (
          <button
            key={m}
            onClick={() => setMode(m)}
            className={cn(
              "rounded-md border px-2.5 py-1",
              mode === m ? "border-accent bg-accent/10 text-accent" : "text-muted-foreground hover:bg-muted",
            )}
          >
            {t(`diff.${m}`)}
          </button>
        ))}
      </div>
      {/* Two headers aligned with the two panes, color-coded so left (earlier)
          vs right (this request) is unmistakable. */}
      <div className="flex gap-2 text-xs">
        <div className="flex flex-1 items-center gap-2 rounded-lg border bg-muted/40 px-3 py-1.5">
          <History size={14} className="shrink-0 text-muted-foreground" />
          <span className="shrink-0 font-semibold text-muted-foreground">{t("diff.earlier")}</span>
          <select
            value={effectiveLeft ?? ""}
            onChange={(e) => setLeftId(e.target.value)}
            className="min-w-0 flex-1 rounded-md border bg-card px-2 py-0.5"
          >
            {olderCompletions.map((e) => (
              <option key={e.id} value={e.id}>
                {fmt(e.started_at)} · {e.model || e.provider}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-1 items-center gap-2 rounded-lg border-2 border-accent bg-accent/10 px-3 py-1.5">
          <CircleDot size={14} className="shrink-0 text-accent" />
          <span className="shrink-0 font-semibold text-accent">{t("diff.current")}</span>
          {current && (
            <span className="truncate text-muted-foreground">
              {fmt(current.started_at)} · {current.model || current.provider}
            </span>
          )}
        </div>
      </div>
      {mode === "semantic" ? (
        effectiveLeft && <SemanticDiff leftId={effectiveLeft} rightId={eventId} />
      ) : left.isLoading || right.isLoading ? (
        <p className="text-xs text-muted-foreground">{t("raw.loading")}</p>
      ) : (
        <Merge left={pretty(left.data ?? "")} right={pretty(right.data ?? "")} />
      )}
    </div>
  );
}

// Merge mounts a CodeMirror MergeView imperatively (no React wrapper needed).
function Merge({ left, right }: { left: string; right: string }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!ref.current) return;
    const ext = [lineNumbers(), json(), EditorView.editable.of(false), EditorState.readOnly.of(true)];
    const view = new MergeView({
      a: { doc: left, extensions: ext },
      b: { doc: right, extensions: ext },
      parent: ref.current,
      collapseUnchanged: { margin: 3, minSize: 6 },
      highlightChanges: true,
      gutter: true,
    });
    return () => view.destroy();
  }, [left, right]);
  return <div ref={ref} className="overflow-auto rounded-lg border text-xs [&_.cm-mergeView]:max-h-[520px]" />;
}

function pretty(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

function fmt(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}
