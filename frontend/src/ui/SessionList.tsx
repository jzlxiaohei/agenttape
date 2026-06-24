import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ChevronRight, Trash2, Globe, Pencil, Check, X } from "lucide-react";
import { httpOnlySession } from "@/api/sessions";
import { useSessionGroupsView, type SessionVM } from "@/viewmodel/sessions";
import { useSessionRoute } from "@/viewmodel/route";
import { useDeleteSession, useRenameSession } from "@/query/sessions";
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
  const filter = useUIStore((s) => s.sessionFilter);
  const setFilter = useUIStore((s) => s.setSessionFilter);

  return (
    <div className="space-y-1">
      <input
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        placeholder={t("sessions.filter_placeholder")}
        className="mb-1 w-full rounded-md border bg-card px-2 py-1 text-xs outline-none focus:border-accent"
      />
      {isLoading ? (
        <p className="px-3 py-2 text-sm text-muted-foreground">{t("sessions.loading")}</p>
      ) : isError ? (
        <p className="px-3 py-2 text-sm text-error">{t("sessions.error")}</p>
      ) : groups.length === 0 ? (
        <p className="px-3 py-2 text-sm leading-relaxed text-muted-foreground">
          {filter ? t("sessions.no_match") : t("sessions.empty")}
        </p>
      ) : (
        groups.map((g) => <ClientGroup key={g.client} client={g.client} sessions={g.sessions} />)
      )}
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
  const rename = useRenameSession();
  const navigate = useNavigate();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");

  // Prefer the user-set / auto-derived name; fall back to a short id when unnamed.
  const displayName = session.title || `#${session.id.slice(0, 8)}`;

  const startEdit = () => {
    setDraft(session.title);
    setEditing(true);
  };
  const save = () => {
    rename.mutate({ id: session.id, title: draft.trim() });
    setEditing(false);
  };

  const remove = () =>
    del.mutate(session.id, {
      // If the open session was the one deleted, fall back to the bare list.
      onSuccess: () => session.selected && navigate("/sessions"),
    });

  if (editing) {
    return (
      <li className="flex items-center gap-1 rounded-md py-1 pl-8 pr-2" onClick={(e) => e.stopPropagation()}>
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") save();
            if (e.key === "Escape") setEditing(false);
          }}
          placeholder={t("sessions.rename_placeholder")}
          className="min-w-0 flex-1 rounded border bg-card px-1.5 py-1 text-xs outline-none focus:border-accent"
        />
        <button onClick={save} title={t("sessions.rename_save")} className="shrink-0 rounded p-1 text-muted-foreground hover:text-accent">
          <Check size={13} />
        </button>
        <button onClick={() => setEditing(false)} title={t("replay.cancel")} className="shrink-0 rounded p-1 text-muted-foreground hover:text-foreground">
          <X size={13} />
        </button>
      </li>
    );
  }

  return (
    <li
      onClick={onSelect}
      title={`#${session.id}`}
      className={cn(
        "group flex cursor-pointer items-center gap-2 rounded-md py-1.5 pl-8 pr-2 transition-colors hover:bg-muted/60",
        session.selected && "bg-accent/10",
      )}
    >
      <span className="shrink-0 text-[11px] text-muted-foreground/50">{ordinal}</span>
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-[13px]",
          session.title ? "" : "mono",
          session.selected ? "font-medium text-foreground" : "text-foreground/90",
        )}
      >
        {displayName}
      </span>
      {httpOnlySession(session) && (
        <Tooltip content={t("sessions.http_only_hint")}>
          <span className="shrink-0 text-muted-foreground/70">
            <Globe size={12} />
          </span>
        </Tooltip>
      )}
      {/* Time keeps its layout width (fades on hover) while the rename + delete
          buttons overlay on top, so the http-only icon to the left never shifts. */}
      <span className="relative flex shrink-0 items-center justify-end">
        <span className="text-xs text-muted-foreground transition-opacity group-hover:opacity-0">
          {timeAgo(session.started_at, i18n.language)}
        </span>
        <span className="pointer-events-none absolute right-0 top-1/2 flex -translate-y-1/2 items-center gap-0.5 opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100">
          <button
            onClick={(e) => {
              e.stopPropagation();
              startEdit();
            }}
            title={t("sessions.rename")}
            className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-accent"
          >
            <Pencil size={12} />
          </button>
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
                className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-error"
              >
                <Trash2 size={13} />
              </button>
            )}
          </Popconfirm>
        </span>
      </span>
    </li>
  );
}
