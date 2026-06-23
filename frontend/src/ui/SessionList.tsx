import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ChevronRight, Trash2, Globe } from "lucide-react";
import { httpOnlySession } from "@/api/sessions";
import { useSessionGroupsView, type SessionVM } from "@/viewmodel/sessions";
import { useSessionRoute } from "@/viewmodel/route";
import { useDeleteSession } from "@/query/sessions";
import { useUIStore } from "@/store/ui";
import { cn } from "@/lib/utils";
import { timeAgo } from "@/lib/time";
import { ClientIcon } from "./ClientIcon";
import { Popconfirm } from "./Popconfirm";
import { Tooltip } from "@/components/ui/tooltip";

// Sidebar session list: collapsible per-client groups (claude_code / codex_cli),
// each row a session with a relative "x ago". Container only — grouping lives in
// the viewmodel, collapse state in the store.
export function SessionList() {
  const { t } = useTranslation();
  const { groups, isLoading, isError } = useSessionGroupsView();

  if (isLoading) return <p className="px-3 py-2 text-sm text-muted-foreground">{t("sessions.loading")}</p>;
  if (isError) return <p className="px-3 py-2 text-sm text-error">{t("sessions.error")}</p>;
  if (groups.length === 0)
    return <p className="px-3 py-2 text-sm leading-relaxed text-muted-foreground">{t("sessions.empty")}</p>;

  return (
    <div className="space-y-1">
      {groups.map((g) => (
        <ClientGroup key={g.client} client={g.client} sessions={g.sessions} />
      ))}
    </div>
  );
}

function ClientGroup({ client, sessions }: { client: string; sessions: SessionVM[] }) {
  const { t } = useTranslation();
  const key = `sidebar-group-${client}`;
  const collapsed = useUIStore((s) => s.collapsed[key]) ?? false;
  const toggle = useUIStore((s) => s.toggleCollapsed);
  const select = useSessionRoute().openSession;

  return (
    <section>
      <button
        onClick={() => toggle(key)}
        className="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-muted/60"
      >
        <ChevronRight
          size={14}
          className={cn("shrink-0 text-muted-foreground transition-transform", !collapsed && "rotate-90")}
        />
        <ClientIcon client={client} size={14} />
        <span className="text-xs font-semibold text-muted-foreground">
          {t(`client.${client}`, { defaultValue: client })}
        </span>
        <span className="text-xs text-muted-foreground/60">{sessions.length}</span>
      </button>
      {!collapsed && (
        <ul>
          {sessions.map((s, index) => (
            <SessionRow key={s.id} session={s} ordinal={index + 1} onSelect={() => select(s.id)} />
          ))}
        </ul>
      )}
    </section>
  );
}

function SessionRow({ session, ordinal, onSelect }: { session: SessionVM; ordinal: number; onSelect: () => void }) {
  const { t, i18n } = useTranslation();
  const del = useDeleteSession();
  const navigate = useNavigate();
  const label = `${ordinal} · #${session.id.slice(0, 8)}`;

  const remove = () =>
    del.mutate(session.id, {
      // If the open session was the one deleted, fall back to the bare list.
      onSuccess: () => session.selected && navigate("/sessions"),
    });

  return (
    <li
      onClick={onSelect}
      title={`#${session.id}`}
      className={cn(
        "group flex cursor-pointer items-center gap-2 rounded-md py-1.5 pl-8 pr-2 transition-colors hover:bg-muted/60",
        session.selected && "bg-accent/10",
      )}
    >
      <span
        className={cn(
          "mono min-w-0 flex-1 truncate text-[13px]",
          session.selected ? "font-medium text-foreground" : "text-foreground/90",
        )}
      >
        {label}
      </span>
      {httpOnlySession(session) && (
        <Tooltip content={t("sessions.http_only_hint")}>
          <span className="shrink-0 text-muted-foreground/70">
            <Globe size={12} />
          </span>
        </Tooltip>
      )}
      {/* Time + delete share one slot: time keeps its layout width (only fades on
          hover) while the delete button is overlaid on top, so the http-only icon
          to the left never shifts. */}
      <span className="relative flex shrink-0 items-center justify-end">
        <span className="text-xs text-muted-foreground transition-opacity group-hover:opacity-0">
          {timeAgo(session.started_at, i18n.language)}
        </span>
        <Popconfirm
          message={t("sessions.delete_confirm", { count: session.event_count })}
          confirmLabel={t("sessions.delete")}
          cancelLabel={t("replay.cancel")}
          tone="danger"
          placement="right"
          onConfirm={remove}
        >
          {({ open }) => (
            <button
              onClick={(e) => {
                e.stopPropagation();
                open();
              }}
              title={t("sessions.delete")}
              className="pointer-events-none absolute right-0 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-error group-hover:pointer-events-auto group-hover:opacity-100"
            >
              <Trash2 size={13} />
            </button>
          )}
        </Popconfirm>
      </span>
    </li>
  );
}
