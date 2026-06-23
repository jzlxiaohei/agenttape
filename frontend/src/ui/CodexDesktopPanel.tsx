import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { Loader2, MonitorDown, RotateCcw, AlertTriangle } from "lucide-react";
import { useCodexDesktopStatus, useInstallCodexDesktop, useRestoreCodexDesktop } from "@/query/codexDesktop";

// Codex desktop launch: there is no terminal to spawn. We write ~/.codex/config.toml
// (backed up first), the user (re)starts the desktop app, then restores when done.
export function CodexDesktopPanel() {
  const { t } = useTranslation();
  const { data: status, isLoading } = useCodexDesktopStatus();
  const install = useInstallCodexDesktop();
  const restore = useRestoreCodexDesktop();
  const [hooks, setHooks] = useState(true);

  if (isLoading) return <p className="text-sm text-muted-foreground">{t("detail.loading")}</p>;

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2 rounded-lg border border-suspected/40 bg-suspected/5 px-3 py-2 text-xs text-suspected">
        <AlertTriangle size={14} className="mt-0.5 shrink-0" />
        <span>{t("launch.desktop.warning")}</span>
      </div>

      {status?.active ? (
        <div className="space-y-3">
          <div className="space-y-2 rounded-md border border-toolcall/40 bg-toolcall/5 px-3 py-2 text-xs text-toolcall">
            <p className="font-medium">{t("launch.desktop.active")}</p>
            <ol className="list-decimal space-y-0.5 pl-4">
              <li>{t("launch.desktop.step_restart")}</li>
              {status.hooks && <li>{t("launch.desktop.step_trust")}</li>}
              <li>{t("launch.desktop.step_done")}</li>
            </ol>
            {status.config_path && <p className="mono text-[11px] opacity-80">{status.config_path}</p>}
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => restore.mutate()}
              disabled={restore.isPending}
              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
            >
              {restore.isPending ? <Loader2 size={15} className="animate-spin" /> : <RotateCcw size={15} />}
              {t("launch.desktop.restore")}
            </button>
            <Link to="/sessions" className="text-xs text-accent underline">{t("launch.view_sessions")}</Link>
          </div>
          {restore.isError && <ErrLine msg={(restore.error as Error).message} />}
        </div>
      ) : (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">{t("launch.desktop.intro")}</p>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={hooks} onChange={(e) => setHooks(e.target.checked)} />
            {t("launch.desktop.hooks")}
          </label>
          <p className="text-xs text-muted-foreground">
            {hooks ? t("launch.desktop.hooks_note") : t("launch.desktop.http_only_note")}
          </p>
          <button
            onClick={() => install.mutate(hooks)}
            disabled={install.isPending}
            className="inline-flex items-center gap-1.5 rounded-md bg-accent px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
          >
            {install.isPending ? <Loader2 size={15} className="animate-spin" /> : <MonitorDown size={15} />}
            {t("launch.desktop.install")}
          </button>
          {install.isError && <ErrLine msg={(install.error as Error).message} />}
        </div>
      )}
    </div>
  );
}

function ErrLine({ msg }: { msg: string }) {
  return <p className="rounded-md border border-error/40 bg-error/5 px-3 py-2 text-xs text-error">{msg}</p>;
}
