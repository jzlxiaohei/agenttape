import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { AlertTriangle, FlaskConical, Loader2, Play } from "lucide-react";
import { useCases, useActiveSessions, useRunCase } from "@/query/cases";
import type { ReplayCase } from "@/api/cases";
import { cn } from "@/lib/utils";
import CodeEditor from "./CodeEditor";

// Replay library: predefined + user-saved cases. Pick a case, pick a session to
// supply credentials, (optionally) edit the body, and run it against upstream.
export function CasesPanel() {
  const { t } = useTranslation();
  const { data: cases, isLoading } = useCases();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = cases ?? [];
  const selected = list.find((c) => c.id === selectedId) ?? list[0] ?? null;

  return (
    <div className="flex h-full">
      <div className="w-72 shrink-0 overflow-y-auto border-r bg-surface">
        <header className="flex items-center gap-2 px-4 py-3">
          <FlaskConical size={16} className="text-accent" />
          <h1 className="text-base font-semibold">{t("cases.title")}</h1>
        </header>
        {isLoading ? (
          <p className="p-4 text-muted-foreground">{t("detail.loading")}</p>
        ) : list.length === 0 ? (
          <p className="p-4 text-sm text-muted-foreground">{t("cases.empty")}</p>
        ) : (
          <ul className="divide-y">
            {list.map((c) => (
              <li
                key={c.id}
                onClick={() => setSelectedId(c.id)}
                className={cn(
                  "cursor-pointer px-4 py-3 transition-colors hover:bg-muted/60",
                  selected?.id === c.id && "bg-accent/8",
                )}
              >
                <div className="truncate text-sm font-medium">{c.name}</div>
                <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="mono">{c.provider}</span>
                  <span
                    className={cn(
                      "rounded px-1.5 py-0.5 text-[10px]",
                      c.source === "seed" ? "bg-reasoning/10 text-reasoning" : "bg-muted text-muted-foreground",
                    )}
                  >
                    {t(`cases.source_${c.source}`, { defaultValue: c.source })}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
      <main className="min-w-0 flex-1 overflow-y-auto">
        {selected ? <CaseRunner caseItem={selected} /> : <Centered>{t("cases.subtitle")}</Centered>}
      </main>
    </div>
  );
}

function CaseRunner({ caseItem }: { caseItem: ReplayCase }) {
  const { t } = useTranslation();
  const { data: sessions } = useActiveSessions();
  const run = useRunCase(caseItem.id);
  const active = sessions ?? [];

  const [sessionId, setSessionId] = useState("");
  const [draft, setDraft] = useState<string | null>(null);
  const [armed, setArmed] = useState(false);

  const effectiveSession = sessionId || active[0]?.id || "";
  const pretty = useMemo(() => prettyJSON(caseItem.body), [caseItem.body]);
  const body = draft ?? pretty;
  const edited = draft !== null && draft !== pretty;
  const validJSON = useMemo(() => isValidJSON(body), [body]);

  const send = () => {
    if (!effectiveSession) return;
    if (!armed) {
      setArmed(true);
      return;
    }
    setArmed(false);
    run.mutate({ sessionId: effectiveSession, body: edited ? body : undefined });
  };

  const result = run.data;

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-6">
      <header className="space-y-1">
        <h2 className="text-lg font-semibold">{caseItem.name}</h2>
        <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
          <span className="mono">{caseItem.method} {caseItem.target}</span>
          {caseItem.tags && <span>· {caseItem.tags}</span>}
        </div>
      </header>

      <div className="flex items-start gap-2 rounded-lg border border-suspected/40 bg-suspected/5 px-3 py-2 text-xs text-suspected">
        <AlertTriangle size={14} className="mt-0.5 shrink-0" />
        <span>{t("cases.warning")}</span>
      </div>

      <label className="block space-y-1">
        <span className="text-xs font-medium text-muted-foreground">{t("cases.session")}</span>
        {active.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            {t("cases.none_active")} <Link to="/launch" className="text-accent underline">{t("launch.title")}</Link>
          </p>
        ) : (
          <select
            value={effectiveSession}
            onChange={(e) => setSessionId(e.target.value)}
            className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
          >
            {active.map((s) => (
              <option key={s.id} value={s.id}>
                {s.client} · {s.upstream} · {s.id.slice(0, 8)}
              </option>
            ))}
          </select>
        )}
      </label>

      <div className="space-y-1">
        <span className="text-xs font-medium text-muted-foreground">
          {t("replay.body")} {edited && <span className="text-accent">· {t("replay.edited")}</span>}
        </span>
        <CodeEditor value={body} onChange={setDraft} height="220px" />
        {!validJSON && (
          <p className="flex items-center gap-1 text-xs text-error">
            <AlertTriangle size={12} /> {t("replay.invalid_json")}
          </p>
        )}
      </div>

      <div className="flex items-center gap-3">
        <button
          onClick={send}
          disabled={run.isPending || !effectiveSession}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50",
            armed ? "bg-error" : "bg-accent",
          )}
        >
          {run.isPending ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
          {run.isPending ? t("replay.sending") : armed ? t("replay.confirm") : t("cases.run")}
        </button>
        {armed && !run.isPending && (
          <button onClick={() => setArmed(false)} className="text-xs text-muted-foreground hover:text-foreground">
            {t("replay.cancel")}
          </button>
        )}
      </div>

      {run.isError && (
        <p className="rounded-md border border-error/40 bg-error/5 px-3 py-2 text-xs text-error">
          {(run.error as Error).message}
        </p>
      )}
      {result && (
        <section className="space-y-2">
          <div className="flex flex-wrap items-center gap-x-3 text-xs text-muted-foreground">
            <span>{t("detail.status")}: {result.status}</span>
            <span>{t("detail.duration")}: {result.duration_ms}ms</span>
            {result.normalized?.response?.usage && (
              <span className="mono">
                {t("detail.usage")}: out {result.normalized.response.usage.output_tokens ?? 0}
              </span>
            )}
          </div>
          {result.normalize_error && <p className="text-xs text-error">{result.normalize_error}</p>}
          <div className="rounded-lg border border-accent/50">
            <div className="border-b px-3 py-1.5 text-xs font-semibold text-accent">{t("replay.result")}</div>
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-words px-3 py-2 text-xs leading-relaxed">
              {result.normalized?.response?.final_text || "—"}
            </pre>
          </div>
        </section>
      )}
    </div>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div className="flex h-full items-center justify-center p-6 text-muted-foreground">{children}</div>;
}

function prettyJSON(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

function isValidJSON(text: string): boolean {
  if (text.trim() === "") return true;
  try {
    JSON.parse(text);
    return true;
  } catch {
    return false;
  }
}
