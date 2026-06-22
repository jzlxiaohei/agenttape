import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { Rocket, Loader2, Terminal, Copy, Check } from "lucide-react";
import { useLaunch, useTerminals, useLaunchPreview } from "@/query/launch";
import type { LaunchKind, LaunchMode } from "@/api/launch";
import { cn } from "@/lib/utils";

// Launch page: start cc/codex through the proxy with one click. Subscription mode
// uses the account you're logged into; key mode keeps the API key in server memory
// only (proxy-injected). codex desktop is a planned follow-up.
export function LaunchPanel() {
  const { t } = useTranslation();
  const [kind, setKind] = useState<LaunchKind>("cc");
  const [mode, setMode] = useState<LaunchMode>("subscription");
  const [workdir, setWorkdir] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [terminal, setTerminal] = useState("");
  const { data: terminals } = useTerminals();
  const launch = useLaunch();

  const termList = terminals ?? [];
  const effectiveTerminal = terminal || termList[0] || "Terminal";

  const preview = useLaunchPreview({ kind, mode, workdir: workdir.trim() || undefined });
  const command = preview.data?.command ?? "";
  const serverLaunchEnabled = preview.data?.enabled ?? false;
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(command).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };

  const canLaunch = mode === "subscription" || apiKey.trim() !== "";

  return (
    <div className="mx-auto max-w-xl space-y-6 p-8">
      <header className="space-y-1">
        <h1 className="flex items-center gap-2 text-lg font-semibold">
          <Rocket size={18} className="text-accent" />
          {t("launch.title")}
        </h1>
        <p className="text-sm text-muted-foreground">{t("launch.subtitle")}</p>
      </header>

      <Field label={t("launch.client")}>
        <div className="grid grid-cols-2 gap-3">
          {(["cc", "codex"] as const).map((k) => (
            <button
              key={k}
              onClick={() => setKind(k)}
              className={cn(
                "rounded-xl border p-4 text-left transition-colors hover:bg-muted/50",
                kind === k ? "border-accent bg-accent/8" : "border-border",
              )}
            >
              <div className="font-medium">{t(`client.${k === "cc" ? "claude_code" : "codex_cli"}`)}</div>
              <div className="mt-0.5 text-xs text-muted-foreground">{k === "cc" ? "claude" : "codex"}</div>
            </button>
          ))}
        </div>
      </Field>

      <Field label={t("launch.credential")}>
        <div className="flex gap-1 text-sm">
          {(["subscription", "key"] as const).map((m) => (
            <button
              key={m}
              onClick={() => setMode(m)}
              className={cn(
                "flex-1 rounded-md border px-3 py-1.5",
                mode === m ? "border-accent bg-accent/10 text-accent" : "text-muted-foreground hover:bg-muted",
              )}
            >
              {t(`launch.mode_${m}`)}
            </button>
          ))}
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          {mode === "subscription" ? t("launch.mode_subscription_note") : t("launch.mode_key_note")}
        </p>
      </Field>

      {mode === "key" && (
        <Field label={t("launch.api_key")}>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="sk-…"
            className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm mono"
          />
        </Field>
      )}

      <Field label={t("launch.workdir")}>
        <input
          value={workdir}
          onChange={(e) => setWorkdir(e.target.value)}
          placeholder={t("launch.workdir_hint")}
          className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm mono"
        />
      </Field>

      {termList.length > 0 && (
        <Field label={t("launch.terminal")}>
          <select
            value={effectiveTerminal}
            onChange={(e) => setTerminal(e.target.value)}
            className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
          >
            {termList.map((tm) => (
              <option key={tm} value={tm}>
                {tm}
              </option>
            ))}
          </select>
        </Field>
      )}

      <Field label={t("launch.run_yourself")}>
        <div className="flex items-stretch gap-2">
          <code className="min-w-0 flex-1 overflow-x-auto whitespace-pre rounded-md border bg-muted px-2.5 py-2 text-xs mono">
            {command || "…"}
          </code>
          <button
            onClick={copy}
            disabled={!command}
            className="inline-flex shrink-0 items-center gap-1 rounded-md border px-2 text-xs text-muted-foreground hover:bg-muted disabled:opacity-50"
          >
            {copied ? <Check size={13} className="text-toolcall" /> : <Copy size={13} />}
            {copied ? t("launch.copied") : t("launch.copy")}
          </button>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">{t("launch.run_yourself_note")}</p>
      </Field>

      <div className="space-y-1">
        <button
          onClick={() =>
            launch.mutate({
              kind,
              mode,
              workdir: workdir.trim() || undefined,
              api_key: mode === "key" ? apiKey.trim() : undefined,
              terminal: effectiveTerminal,
            })
          }
          disabled={launch.isPending || !canLaunch || !serverLaunchEnabled}
          className="inline-flex items-center gap-2 rounded-md bg-accent px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {launch.isPending ? <Loader2 size={15} className="animate-spin" /> : <Terminal size={15} />}
          {launch.isPending ? t("launch.launching") : t("launch.button")}
        </button>
        {!serverLaunchEnabled && <p className="text-xs text-muted-foreground">{t("launch.disabled_note")}</p>}
      </div>

      {launch.isError && (
        <p className="rounded-md border border-error/40 bg-error/5 px-3 py-2 text-xs text-error">
          {(launch.error as Error).message}
        </p>
      )}
      {launch.isSuccess && (
        <div className="space-y-1 rounded-md border border-toolcall/40 bg-toolcall/5 px-3 py-2 text-xs text-toolcall">
          <p>{t("launch.started")}</p>
          <Link to="/sessions" className="underline">
            {t("launch.view_sessions")}
          </Link>
        </div>
      )}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
      {children}
    </div>
  );
}
