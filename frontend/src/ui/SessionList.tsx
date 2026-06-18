import { useTranslation } from "react-i18next";
import { useSessionsView, type SessionVM } from "@/viewmodel/sessions";
import { useUIStore } from "@/store/ui";
import { cn } from "@/lib/utils";

// Container: reads the viewmodel + store, renders. No fetch, no derivation.
export function SessionList() {
  const { t } = useTranslation();
  const { sessions, isLoading, isError } = useSessionsView();
  const select = useUIStore((s) => s.selectSession);

  if (isLoading) return <p className="p-4 text-muted-foreground">{t("sessions.loading")}</p>;
  if (isError) return <p className="p-4 text-error">{t("sessions.error")}</p>;
  if (sessions.length === 0)
    return <p className="p-4 text-sm leading-relaxed text-muted-foreground">{t("sessions.empty")}</p>;

  return (
    <ul>
      {sessions.map((s) => (
        <SessionRow key={s.id} session={s} onSelect={() => select(s.id)} />
      ))}
    </ul>
  );
}

const clientColor: Record<string, string> = {
  claude_code: "bg-[hsl(24_85%_55%)]",
  codex_cli: "bg-[hsl(160_50%_42%)]",
};

// Presentational: a conversation-list style row (avatar + name + preview + meta).
function SessionRow({ session, onSelect }: { session: SessionVM; onSelect: () => void }) {
  const { t } = useTranslation();
  const clientLabel = t(`client.${session.client}`, { defaultValue: session.client });
  const initial = clientLabel.charAt(0).toUpperCase();
  return (
    <li
      onClick={onSelect}
      className={cn(
        "flex cursor-pointer items-center gap-3 px-4 py-3 transition-colors hover:bg-muted/60",
        session.selected && "bg-accent/8",
      )}
    >
      <div
        className={cn(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white",
          clientColor[session.client] ?? "bg-muted-foreground",
        )}
      >
        {initial}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate font-medium">{clientLabel}</span>
          <span className="shrink-0 text-xs text-muted-foreground">
            {t("sessions.events", { count: session.event_count })}
          </span>
        </div>
        <div className="truncate text-xs text-muted-foreground mono">{session.upstream}</div>
      </div>
    </li>
  );
}
