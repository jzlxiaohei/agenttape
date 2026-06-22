import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertTriangle, Play, Loader2 } from "lucide-react";
import { useEventDetail, useRawFile, useReplay } from "@/query/events";
import { useAddCase } from "@/query/cases";
import { cn } from "@/lib/utils";
import CodeEditor from "./CodeEditor";

// Replay tab: re-send this request to upstream (optionally edited) and compare the
// fresh result against the original, side by side. A real billed call, so sending
// is explicit and confirmed; the result is not persisted (next.md 6.2 experiment).
export default function ReplayView({ eventId }: { eventId: string }) {
  const { t } = useTranslation();
  const original = useEventDetail(eventId);
  const raw = useRawFile(eventId, "request_body", true);
  const replay = useReplay(eventId);
  const addCase = useAddCase();
  const [draft, setDraft] = useState<string | null>(null);
  const [armed, setArmed] = useState(false);

  const prettyOriginal = useMemo(() => prettyJSON(raw.data ?? ""), [raw.data]);
  const body = draft ?? prettyOriginal;
  const edited = draft !== null && draft !== prettyOriginal;
  const validJSON = useMemo(() => isValidJSON(body), [body]);

  const send = () => {
    if (!armed) {
      setArmed(true);
      return;
    }
    setArmed(false);
    // undefined = resend verbatim; only send a body when the user edited it
    replay.mutate(edited ? body : undefined);
  };

  const origText = original.data?.normalized?.response?.final_text ?? "";
  const result = replay.data;

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2 rounded-lg border border-suspected/40 bg-suspected/5 px-3 py-2 text-xs text-suspected">
        <AlertTriangle size={14} className="mt-0.5 shrink-0" />
        <span>{t("replay.warning")}</span>
      </div>

      <section className="space-y-1">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t("replay.body")} {edited && <span className="text-accent">· {t("replay.edited")}</span>}
          </h3>
          <div className="flex items-center gap-3">
            {draft !== null && (
              <button onClick={() => setDraft(null)} className="text-xs text-muted-foreground hover:text-foreground">
                {t("replay.reset")}
              </button>
            )}
            <button
              onClick={() => addCase.mutate({ eventId })}
              disabled={addCase.isPending}
              className="text-xs text-accent hover:underline disabled:opacity-50"
            >
              {addCase.isSuccess ? t("replay.save_done") : t("replay.save_case")}
            </button>
          </div>
        </div>
        {raw.isLoading ? (
          <p className="text-xs text-muted-foreground">{t("raw.loading")}</p>
        ) : (
          <CodeEditor value={body} onChange={setDraft} height="280px" />
        )}
        {!validJSON && (
          <p className="flex items-center gap-1 text-xs text-error">
            <AlertTriangle size={12} /> {t("replay.invalid_json")}
          </p>
        )}
      </section>

      <div className="flex items-center gap-3">
        <button
          onClick={send}
          disabled={replay.isPending || raw.isLoading}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50",
            armed ? "bg-error" : "bg-accent",
          )}
        >
          {replay.isPending ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
          {replay.isPending ? t("replay.sending") : armed ? t("replay.confirm") : t("replay.send")}
        </button>
        {armed && !replay.isPending && (
          <button onClick={() => setArmed(false)} className="text-xs text-muted-foreground hover:text-foreground">
            {t("replay.cancel")}
          </button>
        )}
      </div>

      {replay.isError && (
        <p className="rounded-md border border-error/40 bg-error/5 px-3 py-2 text-xs text-error">
          {(replay.error as Error).message}
        </p>
      )}

      {result && (
        <section className="space-y-2">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <span>{t("detail.status")}: {result.status}</span>
            <span>{t("detail.duration")}: {result.duration_ms}ms</span>
            {result.normalized?.response?.usage && (
              <span className="mono">
                {t("detail.usage")}: in {result.normalized.response.usage.input_tokens ?? 0} · out{" "}
                {result.normalized.response.usage.output_tokens ?? 0}
              </span>
            )}
          </div>
          {result.normalize_error && (
            <p className="text-xs text-error">{result.normalize_error}</p>
          )}
          <div className="grid grid-cols-2 gap-3">
            <ResultPane title={t("replay.original")} text={origText} />
            <ResultPane title={t("replay.result")} text={result.normalized?.response?.final_text ?? ""} accent />
          </div>
        </section>
      )}
    </div>
  );
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

function ResultPane({ title, text, accent }: { title: string; text: string; accent?: boolean }) {
  return (
    <div className={cn("rounded-lg border", accent && "border-accent/50")}>
      <div className={cn("border-b px-3 py-1.5 text-xs font-semibold", accent ? "text-accent" : "text-muted-foreground")}>
        {title}
      </div>
      <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-words px-3 py-2 text-xs leading-relaxed">
        {text || "—"}
      </pre>
    </div>
  );
}
