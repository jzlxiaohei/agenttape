import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { Rocket, Loader2, Terminal, Copy, Check, Monitor, FolderClock, ChevronRight, ChevronDown } from "lucide-react";
import { useLaunch, useTerminals, useLaunchPreview, useManualTemplate, useGenerateManual } from "@/query/launch";
import type { LaunchKind, LaunchMode } from "@/api/launch";
import { cn } from "@/lib/utils";
import { ClientIcon } from "./ClientIcon";
import { CodexDesktopPanel } from "./CodexDesktopPanel";

// Launch page: start cc / codex CLI through the proxy with one click (subscription
// uses the account you're logged into; key mode keeps the API key in server memory
// only). The third target, Codex desktop, can't be spawned — it routes via a
// backed-up ~/.codex/config.toml edit, handled by CodexDesktopPanel.
type Target = LaunchKind | "codex-desktop";

export function LaunchPanel() {
  const { t } = useTranslation();
  const [target, setTarget] = useState<Target>("cc");
  const kind: LaunchKind = target === "codex-desktop" ? "codex" : target;
  const isDesktop = target === "codex-desktop";
  const [mode, setMode] = useState<LaunchMode>("subscription");
  const [workdir, setWorkdir] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [terminal, setTerminal] = useState("");
  const [args, setArgs] = useState("");
  const { data: terminals } = useTerminals();
  const launch = useLaunch();

  const termList = terminals ?? [];
  const effectiveTerminal = terminal || termList[0] || "Terminal";
  const trimmedArgs = args.trim() || undefined;
  const rememberWorkdir = () => saveWorkdirHistory(workdir);

  const preview = useLaunchPreview({ kind, mode, workdir: workdir.trim() || undefined, args: trimmedArgs });
  const fullCommand = preview.data?.command ?? "";
  const serverLaunchEnabled = preview.data?.enabled ?? false;

  // Primary "run it yourself": the env/-c command. A <TOKEN> template shows live;
  // a real session is registered only when the user copies.
  const manualReq = { kind, mode, args: trimmedArgs };
  const template = useManualTemplate(manualReq);
  const generate = useGenerateManual();
  const [envCmd, setEnvCmd] = useState<string | null>(null); // generated (real) command
  const [envCopied, setEnvCopied] = useState(false);
  useEffect(() => setEnvCmd(null), [kind, mode, args]); // a generated cmd is tied to this config
  const envDisplay = envCmd ?? template.data?.command ?? "";
  const copyEnv = () => {
    rememberWorkdir();
    generate.mutate(manualReq, {
      onSuccess: (res) => {
        setEnvCmd(res.command);
        navigator.clipboard.writeText(res.command).then(() => {
          setEnvCopied(true);
          setTimeout(() => setEnvCopied(false), 1500);
        });
      },
    });
  };

  // Secondary "full capture (http + hooks)": the tracelab launch one-liner.
  const [fullOpen, setFullOpen] = useState(false);
  const [fullCopied, setFullCopied] = useState(false);
  const copyFull = () => {
    navigator.clipboard.writeText(fullCommand).then(() => {
      rememberWorkdir();
      setFullCopied(true);
      setTimeout(() => setFullCopied(false), 1500);
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
        <div className="grid grid-cols-3 gap-3">
          {([
            { id: "cc", label: t("client.claude_code"), sub: "claude" },
            { id: "codex", label: t("client.codex_cli"), sub: "codex" },
            { id: "codex-desktop", label: t("client.codex_desktop"), sub: "codex · desktop" },
          ] as const).map((o) => (
            <button
              key={o.id}
              onClick={() => setTarget(o.id)}
              className={cn(
                "rounded-xl border p-4 text-left transition-colors hover:bg-muted/50",
                target === o.id ? "border-accent bg-accent/8" : "border-border",
              )}
            >
              <div className="flex items-center gap-2 font-medium">
                {o.id === "codex-desktop" ? <Monitor size={16} className="text-codex" /> : <ClientIcon client={o.id} size={16} />}
                <span className="truncate">{o.label}</span>
              </div>
              <div className="mt-0.5 text-xs text-muted-foreground">{o.sub}</div>
            </button>
          ))}
        </div>
      </Field>

      {isDesktop && <CodexDesktopPanel />}
      {!isDesktop && (
        <>{/* cc / codex CLI flow */}

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

      <Field label={t("launch.client_args")}>
        <input
          value={args}
          onChange={(e) => setArgs(e.target.value)}
          placeholder={t("launch.client_args_hint")}
          className="mono w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
        />
        <p className="mt-1 text-xs text-muted-foreground">{t("launch.client_args_note")}</p>
      </Field>

      {/* Two clearly-separated start methods, each badged with what it captures. */}
      <MethodCard title={t("launch.method_spawn")} badge={t("launch.capture_full")} full>
        <p className="text-xs text-muted-foreground">{t("launch.macos_only")}</p>
        <Field label={t("launch.workdir")}>
          <WorkdirInput
            value={workdir}
            onChange={setWorkdir}
            placeholder={t("launch.workdir_hint")}
            recentLabel={t("launch.workdir_recent")}
          />
        </Field>

        {termList.length > 0 && (
          <Field label={t("launch.terminal")}>
            <input
              list="launch-terminal-apps"
              value={effectiveTerminal}
              onChange={(e) => setTerminal(e.target.value)}
              placeholder={t("launch.terminal_hint")}
              className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
            />
            <datalist id="launch-terminal-apps">
              {termList.map((tm) => (
                <option key={tm} value={tm} />
              ))}
            </datalist>
            <p className="mt-1 text-xs text-muted-foreground">{t("launch.terminal_note")}</p>
          </Field>
        )}

        <div className="space-y-1">
          <button
            onClick={() => {
              rememberWorkdir();
              launch.mutate({
                kind,
                mode,
                workdir: workdir.trim() || undefined,
                api_key: mode === "key" ? apiKey.trim() : undefined,
                terminal: effectiveTerminal,
                args: trimmedArgs,
              });
            }}
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
      </MethodCard>

      <MethodCard title={t("launch.run_yourself")} badge={t("launch.capture_http")}>
        <div className="flex items-stretch gap-2">
          <code className="min-w-0 flex-1 overflow-x-auto whitespace-pre rounded-md border bg-muted px-2.5 py-2 text-xs mono">
            {envDisplay || "…"}
          </code>
          <button
            onClick={copyEnv}
            disabled={generate.isPending || !envDisplay}
            className="inline-flex shrink-0 items-center gap-1 rounded-md border px-2 text-xs text-muted-foreground hover:bg-muted disabled:opacity-50"
          >
            {generate.isPending ? (
              <Loader2 size={13} className="animate-spin" />
            ) : envCopied ? (
              <Check size={13} className="text-toolcall" />
            ) : (
              <Copy size={13} />
            )}
            {envCopied ? t("launch.copied") : t("launch.copy")}
          </button>
        </div>
        <p className="text-xs text-muted-foreground">{t("launch.run_yourself_note")}</p>
        <p className="text-xs text-muted-foreground">{t("launch.run_yourself_os")}</p>
        {envCmd && generate.data?.session_id && (
          <p className="text-xs text-toolcall">
            {t("launch.run_yourself_registered", { id: generate.data.session_id.slice(0, 8) })}
          </p>
        )}

        <div>
          <button
            type="button"
            onClick={() => setFullOpen((v) => !v)}
            className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          >
            {fullOpen ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
            {t("launch.full_capture")}
          </button>
          {fullOpen && (
            <div className="mt-1.5 space-y-1">
              <div className="flex items-stretch gap-2">
                <code className="min-w-0 flex-1 overflow-x-auto whitespace-pre rounded-md border bg-muted px-2.5 py-2 text-xs mono">
                  {fullCommand || "…"}
                </code>
                <button
                  onClick={copyFull}
                  disabled={!fullCommand}
                  className="inline-flex shrink-0 items-center gap-1 rounded-md border px-2 text-xs text-muted-foreground hover:bg-muted disabled:opacity-50"
                >
                  {fullCopied ? <Check size={13} className="text-toolcall" /> : <Copy size={13} />}
                  {fullCopied ? t("launch.copied") : t("launch.copy")}
                </button>
              </div>
              <p className="text-xs text-muted-foreground">{t("launch.full_capture_note")}</p>
            </div>
          )}
        </div>
      </MethodCard>
        </>
      )}
    </div>
  );
}

const workdirHistoryKey = "tracelab.launch.workdirs";

function readWorkdirHistory(): string[] {
  try {
    const raw = window.localStorage.getItem(workdirHistoryKey);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((v): v is string => typeof v === "string") : [];
  } catch {
    return [];
  }
}

function saveWorkdirHistory(value: string) {
  const dir = value.trim();
  if (!dir) return;
  const next = [dir, ...readWorkdirHistory().filter((v) => v !== dir)].slice(0, 8);
  window.localStorage.setItem(workdirHistoryKey, JSON.stringify(next));
}

function WorkdirInput({
  value,
  onChange,
  placeholder,
  recentLabel,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  recentLabel: string;
}) {
  const [history, setHistory] = useState<string[]>(() => readWorkdirHistory());
  const [open, setOpen] = useState(false);
  const filtered = useMemo(() => {
    const q = value.trim().toLowerCase();
    if (!q) return history;
    return history.filter((dir) => dir.toLowerCase().includes(q));
  }, [history, value]);

  const refresh = () => setHistory(readWorkdirHistory());
  const remember = () => {
    saveWorkdirHistory(value);
    refresh();
  };

  return (
    <div className="relative">
      <input
        id="launch-workdir"
        name="workdir"
        autoComplete="on"
        value={value}
        onFocus={() => {
          refresh();
          setOpen(true);
        }}
        onBlur={() => {
          remember();
          window.setTimeout(() => setOpen(false), 120);
        }}
        onChange={(e) => {
          onChange(e.target.value);
          setOpen(true);
        }}
        placeholder={placeholder}
        className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm mono"
      />
      {open && filtered.length > 0 && (
        <div className="absolute left-0 right-0 top-full z-40 mt-1 overflow-hidden rounded-lg border bg-card shadow-lg">
          <div className="flex items-center gap-1.5 border-b px-2.5 py-1.5 text-[11px] font-medium text-muted-foreground">
            <FolderClock size={12} />
            {recentLabel}
          </div>
          <div className="max-h-56 overflow-auto p-1">
            {filtered.map((dir) => (
              <button
                key={dir}
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => {
                  onChange(dir);
                  saveWorkdirHistory(dir);
                  refresh();
                  setOpen(false);
                }}
                className="mono block w-full truncate rounded-md px-2 py-1.5 text-left text-xs hover:bg-muted"
              >
                {dir}
              </button>
            ))}
          </div>
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

// MethodCard frames one start method (server-spawn vs run-yourself) so the two
// paths read as distinct choices, each badged with what it captures (http+hooks
// vs http-only).
function MethodCard({
  title,
  badge,
  full,
  children,
}: {
  title: string;
  badge: string;
  full?: boolean;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-xl border bg-card">
      <header className="flex items-center justify-between gap-2 border-b px-4 py-2.5">
        <h2 className="text-sm font-semibold">{title}</h2>
        <span
          className={cn(
            "mono shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium",
            full ? "bg-accent/10 text-accent" : "bg-muted text-muted-foreground",
          )}
        >
          {badge}
        </span>
      </header>
      <div className="space-y-3 p-4">{children}</div>
    </section>
  );
}
