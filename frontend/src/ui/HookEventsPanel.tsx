import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Webhook, Plus, Trash2, Loader2 } from "lucide-react";
import { useHookEvents, useAddHookEvent, useSetHookEventEnabled, useDeleteHookEvent } from "@/query/hookEvents";
import type { HookEventDef } from "@/api/hookEvents";
import { hookClientGroups, type HookClientGroup } from "@/viewmodel/hookEvents";
import { cn } from "@/lib/utils";
import { ClientIcon } from "./ClientIcon";
import { Popconfirm } from "./Popconfirm";

// Hook capture settings: the per-client set of lifecycle/tool events agenttape
// wires when it launches each coding agent. Seeded with built-in defaults but
// user-editable, so a newly-shipped runtime event can be captured (or a noisy one
// muted) without waiting for a agenttape release. The launch reads the enabled set.
export function HookEventsPanel() {
  const { t } = useTranslation();
  const { data, isLoading } = useHookEvents();
  const groups = useMemo(() => hookClientGroups(data ?? []), [data]);

  return (
    <div className="h-full w-full overflow-y-auto bg-surface">
      <div className="mx-auto max-w-3xl px-6 py-8">
        <header className="mb-2 flex items-center gap-2">
          <Webhook size={18} className="text-accent" />
          <h1 className="text-lg font-semibold">{t("hooks.title")}</h1>
        </header>
        <p className="mb-6 max-w-2xl text-sm text-muted-foreground">{t("hooks.intro")}</p>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t("detail.loading")}</p>
        ) : (
          <div className="space-y-6">
            {groups.map((g) => (
              <ClientHookCard key={g.client} group={g} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function ClientHookCard({ group }: { group: HookClientGroup }) {
  const { t } = useTranslation();
  const toggle = useSetHookEventEnabled();
  const remove = useDeleteHookEvent();
  const add = useAddHookEvent();
  const [draft, setDraft] = useState("");

  const submitAdd = () => {
    const event = draft.trim();
    if (!event) return;
    add.mutate({ client: group.client, event }, { onSuccess: () => setDraft("") });
  };

  return (
    <section className="overflow-hidden rounded-lg border bg-card">
      <header className="flex items-center gap-2 border-b px-4 py-3">
        <ClientIcon client={group.client} size={16} />
        <h2 className="text-sm font-semibold">
          {t(`hooks.client_${group.client}`, { defaultValue: group.client })}
        </h2>
        <span className="ml-auto rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
          {t("hooks.count", { enabled: group.enabledCount, total: group.events.length })}
        </span>
      </header>

      <ul className="divide-y">
        {group.events.map((ev) => (
          <EventRow
            key={ev.event}
            def={ev}
            busy={toggle.isPending || remove.isPending}
            onToggle={(enabled) => toggle.mutate({ client: ev.client, event: ev.event, enabled })}
            onRemove={() => remove.mutate({ client: ev.client, event: ev.event })}
          />
        ))}
      </ul>

      <div className="flex items-center gap-2 border-t px-4 py-3">
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submitAdd()}
          placeholder={t("hooks.add_placeholder")}
          className="mono min-w-0 flex-1 rounded-md border bg-surface px-2.5 py-1.5 text-sm outline-none focus:border-accent"
        />
        <button
          onClick={submitAdd}
          disabled={!draft.trim() || add.isPending}
          className="inline-flex shrink-0 items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted disabled:opacity-50"
        >
          {add.isPending ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
          {t("hooks.add")}
        </button>
      </div>
      {add.isError && <p className="px-4 pb-3 text-xs text-error">{(add.error as Error).message}</p>}
      <p className="px-4 pb-3 text-xs text-muted-foreground">{t("hooks.add_hint")}</p>
    </section>
  );
}

function EventRow({
  def,
  busy,
  onToggle,
  onRemove,
}: {
  def: HookEventDef;
  busy: boolean;
  onToggle: (enabled: boolean) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const isSeed = def.source === "seed";

  return (
    <li className="group flex items-center gap-3 px-4 py-2.5">
      <Switch checked={def.enabled} disabled={busy} onChange={onToggle} />
      <span className={cn("mono flex-1 truncate text-sm", !def.enabled && "text-muted-foreground line-through")}>
        {def.event}
      </span>
      <span
        className={cn(
          "rounded px-1.5 py-0.5 text-[10px]",
          isSeed ? "bg-reasoning/10 text-reasoning" : "bg-accent/10 text-accent",
        )}
      >
        {t(isSeed ? "hooks.source_seed" : "hooks.source_user")}
      </span>
      {isSeed ? (
        <span className="w-6" />
      ) : (
        <Popconfirm
          message={t("hooks.delete_confirm")}
          confirmLabel={t("hooks.delete")}
          cancelLabel={t("replay.cancel")}
          tone="danger"
          placement="right"
          onConfirm={onRemove}
        >
          {({ open }) => (
            <button
              onClick={open}
              title={t("hooks.delete")}
              className="shrink-0 rounded p-1 text-muted-foreground hover:bg-muted hover:text-error"
            >
              <Trash2 size={13} />
            </button>
          )}
        </Popconfirm>
      )}
    </li>
  );
}

function Switch({
  checked,
  disabled,
  onChange,
}: {
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent p-0 transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        checked ? "bg-accent" : "bg-muted",
      )}
    >
      <span
        className={cn(
          "pointer-events-none block h-4 w-4 rounded-full bg-white shadow-sm transition-transform",
          checked ? "translate-x-4" : "translate-x-0",
        )}
      />
    </button>
  );
}
