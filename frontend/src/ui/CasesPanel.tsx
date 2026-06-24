import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import {
  AlertTriangle,
  Check,
  ChevronDown,
  ChevronRight,
  Copy,
  Eye,
  EyeOff,
  FlaskConical,
  Loader2,
  PanelRightOpen,
  Play,
  Plus,
  Save,
  Terminal,
  Trash2,
  X,
} from "lucide-react";
import {
  useCases,
  useActiveSessions,
  useCloseActiveSession,
  useReenterSessionKey,
  useRunCase,
  useSnapshotCase,
  useDeleteCase,
  useCreateCase,
  useOverwriteCase,
  useCaseCurl,
} from "@/query/cases";
import type { ActiveSession, ReplayCase, CurlMode } from "@/api/cases";
import type { ReplayResult } from "@/api/events";
import {
  caseCardMeta,
  caseDescription,
  caseDisplayName,
  caseHint,
  caseEndpoint,
  caseProviderClient,
  caseProviders,
  caseRunURL,
  caseSections,
  authTargets,
  filterCasesByProvider,
  providerMatchesClient,
} from "@/viewmodel/cases";
import { useUIStore } from "@/store/ui";
import { visibleExperiments } from "@/lib/experiments";
import { ExperimentCard } from "./ExperimentCard";
import { cn } from "@/lib/utils";
import { Dialog, DialogA11y, DialogContent } from "@/components/ui/dialog";
import { Sheet, SheetContent, SheetA11y } from "@/components/ui/sheet";
import CodeEditor from "./CodeEditor";
import { ClientIcon } from "./ClientIcon";
import { Popconfirm } from "./Popconfirm";

// Replay library: predefined + user-saved cases, shown as a card gallery grouped
// into built-in / added, with a provider filter on top. Clicking a card opens the
// runner (pick a session for credentials, optionally edit the body, run upstream).
export function CasesPanel() {
  const { t } = useTranslation();
  const { data: cases, isLoading } = useCases();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const provider = useUIStore((s) => s.casesProvider);
  const setProvider = useUIStore((s) => s.setCasesProvider);
  const del = useDeleteCase();

  const list = cases ?? [];
  const providers = useMemo(() => caseProviders(list), [list]);
  const sections = useMemo(
    () => caseSections(filterCasesByProvider(list, provider)),
    [list, provider],
  );
  const exps = useMemo(() => visibleExperiments(provider), [provider]);
  const selected = list.find((c) => c.id === selectedId) ?? null;

  const remove = (c: ReplayCase) =>
    del.mutate(c.id, { onSuccess: () => selectedId === c.id && setSelectedId(null) });

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      <header className="flex flex-wrap items-center gap-3 border-b bg-surface px-6 py-3">
        <FlaskConical size={16} className="text-accent" />
        <div className="min-w-0">
          <h1 className="text-base font-semibold leading-tight">{t("cases.title")}</h1>
          <p className="text-xs text-muted-foreground">{t("cases.subtitle")}</p>
        </div>
        {providers.length > 1 && (
          <ProviderFilter
            providers={providers}
            value={provider}
            onChange={setProvider}
          />
        )}
        <button
          type="button"
          onClick={() => setCreating(true)}
          className="ml-auto inline-flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted"
        >
          <Plus size={13} />
          {t("cases.new")}
        </button>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t("detail.loading")}</p>
        ) : (
          <div className="space-y-7">
            {list.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("cases.empty")}</p>
            ) : sections.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("cases.no_match")}</p>
            ) : (
              sections.map((section) => (
                <GallerySection
                  key={section.key}
                  sectionKey={`cases-${section.key}`}
                  title={t(`cases.section_${section.key}`)}
                  count={section.cases.length}
                >
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    {section.cases.map((c) => (
                      <CaseCard
                        key={c.id}
                        caseItem={c}
                        onOpen={() => setSelectedId(c.id)}
                        onDelete={() => remove(c)}
                      />
                    ))}
                  </div>
                </GallerySection>
              ))
            )}

            {exps.length > 0 && (
              <GallerySection sectionKey="cases-experiments" title={t("experiments.section")} count={exps.length}>
                <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
                  {exps.map((e) => (
                    <ExperimentCard key={e.id} experiment={e} />
                  ))}
                </div>
              </GallerySection>
            )}
          </div>
        )}
      </div>

      <Dialog open={selected !== null} onOpenChange={(open) => !open && setSelectedId(null)}>
        {selected && (
          <DialogContent
            closeLabel={t("common.close")}
            className="h-[min(860px,calc(100vh-48px))]"
            // Popconfirm popovers portal to body (outside this content), so a click
            // on them reads as "interact outside" and would close the dialog. Keep
            // it open when the interaction originates from a Popconfirm.
            onInteractOutside={(e) => {
              const target = e.detail.originalEvent.target as HTMLElement | null;
              if (target?.closest("[data-popconfirm]")) e.preventDefault();
            }}
          >
            <DialogA11y title={caseDisplayName(selected, t)} />
            <CaseRunner key={selected.id} caseItem={selected} onCreated={setSelectedId} />
          </DialogContent>
        )}
      </Dialog>

      <Dialog open={creating} onOpenChange={setCreating}>
        <DialogContent closeLabel={t("common.close")} className="h-[min(820px,calc(100vh-64px))]">
          <DialogA11y title={t("cases.new_title")} />
          <CaseCreator
            onCancel={() => setCreating(false)}
            onCreated={(id) => {
              setSelectedId(id);
              setCreating(false);
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ProviderFilter is a segmented control: "All" + one chip per provider (brand
// icon). Filter state lives in the UI store (frontend-mvvm §1).
function ProviderFilter({
  providers,
  value,
  onChange,
}: {
  providers: string[];
  value: string;
  onChange: (p: string) => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="inline-flex flex-wrap items-center gap-1 rounded-md border bg-card p-0.5 text-xs">
      <FilterChip active={value === ""} onClick={() => onChange("")}>
        {t("cases.filter_all")}
      </FilterChip>
      {providers.map((p) => (
        <FilterChip key={p} active={value === p} onClick={() => onChange(p)}>
          <ClientIcon client={caseProviderClient(p)} size={13} />
          <span className="mono">{p}</span>
        </FilterChip>
      ))}
    </div>
  );
}

function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1.5 rounded px-2.5 py-1 font-medium transition-colors",
        active ? "bg-accent text-white" : "text-muted-foreground hover:text-foreground",
      )}
    >
      {children}
    </button>
  );
}

// GallerySection is a collapsible group header (chevron + title + count). Collapse
// state lives in the shared UI store, keyed per section so it survives navigation.
function GallerySection({
  sectionKey,
  title,
  count,
  children,
}: {
  sectionKey: string;
  title: string;
  count: number;
  children: React.ReactNode;
}) {
  const collapsed = useUIStore((s) => s.collapsed[sectionKey]);
  const toggle = useUIStore((s) => s.toggleCollapsed);
  return (
    <section>
      <button
        type="button"
        onClick={() => toggle(sectionKey)}
        className="mb-3 flex w-full items-center gap-2 text-left"
      >
        <ChevronRight
          size={14}
          className={cn("shrink-0 text-muted-foreground transition-transform", !collapsed && "rotate-90")}
        />
        <h2 className="font-semibold uppercase tracking-wide text-muted-foreground">{title}</h2>
        <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
          {count}
        </span>
      </button>
      {!collapsed && children}
    </section>
  );
}

// CaseCard surfaces a case at a glance: provider (icon), title, the experiment
// blurb, and a few core fields pulled from the body (endpoint, model, tools,
// streaming). The whole card opens the runner; delete is a guarded sub-action.
function CaseCard({
  caseItem,
  onOpen,
  onDelete,
}: {
  caseItem: ReplayCase;
  onOpen: () => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  const meta = caseCardMeta(caseItem);
  const desc = caseDescription(caseItem, t);
  const endpoint = caseEndpoint(caseItem);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onOpen();
        }
      }}
      className="group flex cursor-pointer flex-col gap-3 rounded-lg border bg-card p-4 text-left shadow-sm transition-colors hover:border-accent/50 hover:bg-muted/30"
    >
      <div className="flex items-center gap-2">
        <span className="inline-flex items-center gap-1.5">
          <ClientIcon client={caseProviderClient(caseItem.provider)} size={14} />
          <span className="mono truncate text-xs text-muted-foreground">{caseItem.provider}</span>
        </span>
        <span
          className={cn(
            "rounded px-1.5 py-0.5 text-[10px] font-medium",
            caseItem.source === "seed" ? "bg-reasoning/10 text-reasoning" : "bg-accent/10 text-accent",
          )}
        >
          {t(`cases.source_${caseItem.source}`, { defaultValue: caseItem.source })}
        </span>
        {/* Built-in (seed) cases are not user-deletable; only added cases are. */}
        {caseItem.source !== "seed" && (
          <Popconfirm
            message={t("cases.delete_confirm")}
            confirmLabel={t("cases.delete")}
            cancelLabel={t("replay.cancel")}
            tone="danger"
            placement="right"
            onConfirm={onDelete}
          >
            {({ open }) => (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  open();
                }}
                title={t("cases.delete")}
                className="ml-auto shrink-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-error group-hover:opacity-100"
              >
                <Trash2 size={13} />
              </button>
            )}
          </Popconfirm>
        )}
      </div>

      <div className="min-w-0">
        <h3 className="truncate text-sm font-semibold">{caseDisplayName(caseItem, t)}</h3>
        {desc && <p className="mt-1 line-clamp-2 text-xs leading-relaxed text-muted-foreground">{desc}</p>}
      </div>

      <div className="mt-auto flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
        <span className="mono max-w-full truncate rounded bg-muted/60 px-1.5 py-0.5">
          {caseItem.method} {endpoint}
        </span>
        {meta.model && (
          <span className="mono max-w-full truncate rounded bg-muted/60 px-1.5 py-0.5">{meta.model}</span>
        )}
        {meta.tools > 0 && (
          <span className="rounded bg-muted/60 px-1.5 py-0.5">{t("cases.field_tools", { count: meta.tools })}</span>
        )}
        {meta.stream && <span className="rounded bg-muted/60 px-1.5 py-0.5">{t("cases.field_stream")}</span>}
      </div>

      <CaseTags tags={caseItem.tags} />
    </div>
  );
}

const manualDefaultBody = JSON.stringify({ model: "gpt-5.5", input: "你是谁?", stream: true }, null, 2);

function CaseCreator({ onCancel, onCreated }: { onCancel: () => void; onCreated: (id: string) => void }) {
  const { t } = useTranslation();
  const create = useCreateCase();
  const [name, setName] = useState("");
  const [tags, setTags] = useState("");
  const [provider, setProvider] = useState("openai-responses");
  const [endpoint, setEndpoint] = useState("/responses");
  const [body, setBody] = useState(manualDefaultBody);
  const validJSON = useMemo(() => isValidJSON(body), [body]);

  const save = () => {
    if (!validJSON) return;
    create.mutate(
      { name, tags, provider, endpoint, body },
      { onSuccess: (c) => onCreated(c.id) },
    );
  };

  return (
    <div className="flex h-full w-full min-w-0 flex-col overflow-hidden">
      <header className="border-b px-6 py-3">
        <h2 className="text-base font-semibold">{t("cases.new_title")}</h2>
        <p className="mt-0.5 text-xs text-muted-foreground">{t("cases.new_hint")}</p>
      </header>
      <div className="grid gap-3 border-b px-6 py-3 md:grid-cols-[1fr_180px_180px]">
        <label className="space-y-1">
          <span className="text-xs font-medium text-muted-foreground">{t("cases.name")}</span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("cases.name_hint")}
            className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
          />
        </label>
        <label className="space-y-1">
          <span className="text-xs font-medium text-muted-foreground">{t("cases.provider")}</span>
          <select
            value={provider}
            onChange={(e) => {
              const next = e.target.value;
              setProvider(next);
              setEndpoint(next === "anthropic" ? "/v1/messages" : "/responses");
            }}
            className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
          >
            <option value="openai-responses">openai-responses</option>
            <option value="anthropic">anthropic</option>
          </select>
        </label>
        <label className="space-y-1">
          <span className="text-xs font-medium text-muted-foreground">{t("cases.endpoint")}</span>
          <input
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value)}
            className="mono w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
          />
        </label>
        <label className="space-y-1 md:col-span-3">
          <span className="text-xs font-medium text-muted-foreground">{t("cases.tags")}</span>
          <input
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            placeholder={t("cases.tags_hint")}
            className="w-full rounded-md border bg-card px-2.5 py-1.5 text-sm"
          />
        </label>
      </div>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden px-6 py-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground">{t("replay.body")}</span>
          {!validJSON && (
            <span className="flex items-center gap-1 text-xs text-error">
              <AlertTriangle size={12} /> {t("replay.invalid_json")}
            </span>
          )}
        </div>
        <CodeEditor
          value={body}
          onChange={setBody}
          height="100%"
          className="min-h-0 min-w-0 flex-1"
          showFoldControls
        />
        {create.isError && <p className="text-xs text-error">{(create.error as Error).message}</p>}
      </div>
      <footer className="flex items-center gap-3 border-t px-6 py-3">
        <button
          type="button"
          onClick={save}
          disabled={!validJSON || create.isPending}
          className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {create.isPending ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
          {create.isPending ? t("cases.saving") : t("cases.create")}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted"
        >
          {t("replay.cancel")}
        </button>
      </footer>
    </div>
  );
}

function CaseRunner({ caseItem, onCreated }: { caseItem: ReplayCase; onCreated: (id: string) => void }) {
  const { t } = useTranslation();
  const { data: sessions } = useActiveSessions();
  const run = useRunCase(caseItem.id);
  const snapshot = useSnapshotCase(caseItem.id);
  const overwrite = useOverwriteCase(caseItem.id);
  const active = sessions ?? [];

  const [sessionId, setSessionId] = useState("");
  const [draft, setDraft] = useState<string | null>(null);
  const [resultOpen, setResultOpen] = useState(false);
  const [curlOpen, setCurlOpen] = useState(false);

  // Only sessions whose client matches this case's provider can serve it; among
  // those, distinct (upstream, credential_kind) pairs are the real auth choices.
  const compatible = active.filter((s) => providerMatchesClient(caseItem.provider, s.client));
  const targets = authTargets(compatible);
  const effectiveSession =
    sessionId && compatible.some((s) => s.id === sessionId) ? sessionId : targets[0]?.id ?? "";
  const selectedSession = active.find((s) => s.id === effectiveSession);
  const endpoint = caseEndpoint(caseItem);
  const runURL = caseRunURL(caseItem, selectedSession);
  const pretty = useMemo(() => prettyJSON(caseItem.body), [caseItem.body]);
  const body = draft ?? pretty;
  const edited = draft !== null && draft !== pretty;
  const validJSON = useMemo(() => isValidJSON(body), [body]);

  const doRun = () =>
    run.mutate(
      { sessionId: effectiveSession, body: edited ? body : undefined },
      { onSettled: () => setResultOpen(true) },
    );

  const saveSnapshot = () => {
    if (!edited || !validJSON) return;
    snapshot.mutate({ body }, { onSuccess: (c) => onCreated(c.id) });
  };

  const saveOverwrite = () => {
    if (!edited || !validJSON) return;
    // Reset the draft on success so the editor shows the freshly-saved server copy
    // (and "edited" clears).
    overwrite.mutate(body, { onSuccess: () => setDraft(null) });
  };

  const result = run.data;

  return (
    <div className="flex h-full w-full min-w-0 flex-col overflow-hidden">
      <header className="space-y-2 border-b px-6 py-3 pr-14">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h2 className="truncate text-base font-semibold">{caseDisplayName(caseItem, t)}</h2>
            <div className="mt-0.5 grid min-w-0 gap-1 text-xs text-muted-foreground">
              <span className="mono min-w-0 truncate">
                {t("cases.endpoint")}: {caseItem.method} {endpoint}
              </span>
              <span className="mono min-w-0 truncate">
                {t("cases.original_url")}: {caseItem.target}
              </span>
              {runURL && (
                <span className="mono min-w-0 truncate">
                  {t("cases.run_url")}: {runURL}
                </span>
              )}
              <CaseTags tags={caseItem.tags} />
            </div>
          </div>
          <div className="w-[min(34vw,440px)] shrink-0">
            {compatible.length === 0 ? (
              <p className="text-right text-xs text-muted-foreground">
                {t("cases.none_active")}{" "}
                <Link to="/launch" className="text-accent underline">{t("launch.title")}</Link>
              </p>
            ) : (
              <AuthSelect
                targets={targets}
                value={effectiveSession}
                label={t("cases.session")}
                onChange={setSessionId}
              />
            )}
          </div>
        </div>
      </header>

      {selectedSession?.needs_key && <ReenterKeyBanner session={selectedSession} />}

      <p className="border-b px-6 py-2 text-xs leading-relaxed text-muted-foreground">
        {t("cases.replay_scope_note")}
      </p>
      {caseHint(caseItem, t) && (
        <p className="border-b border-accent/30 bg-accent/5 px-6 py-2 text-xs leading-relaxed text-foreground/80">
          {caseHint(caseItem, t)}
        </p>
      )}

      <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden px-6 py-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground">
            {t("replay.body")} {edited && <span className="text-accent">· {t("replay.edited")}</span>}
          </span>
          {!validJSON && (
            <span className="flex items-center gap-1 text-xs text-error">
              <AlertTriangle size={12} /> {t("replay.invalid_json")}
            </span>
          )}
        </div>
        <CodeEditor
          value={body}
          onChange={setDraft}
          height="100%"
          className="min-h-0 min-w-0 flex-1"
          showFoldControls
        />
        {snapshot.isError && (
          <p className="text-xs text-error">{(snapshot.error as Error).message}</p>
        )}
      </div>

      <footer className="flex items-center gap-3 border-t px-6 py-3">
        <Popconfirm
          message={t("cases.warning")}
          confirmLabel={t("replay.confirm")}
          cancelLabel={t("replay.cancel")}
          tone="danger"
          disabled={run.isPending || !effectiveSession}
          onConfirm={doRun}
        >
          {({ open }) => (
            <button
              onClick={open}
              disabled={run.isPending || !effectiveSession}
              className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            >
              {run.isPending ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
              {run.isPending ? t("replay.sending") : t("cases.run")}
            </button>
          )}
        </Popconfirm>

        {/* Overwrite + snapshot stay visible (so they're discoverable) but are
            disabled until the body is actually edited — there's nothing to save
            otherwise. */}
        <Popconfirm
          message={t("cases.overwrite_confirm")}
          confirmLabel={t("cases.save_overwrite")}
          cancelLabel={t("replay.cancel")}
          tone="danger"
          disabled={!edited || overwrite.isPending || !validJSON}
          onConfirm={saveOverwrite}
        >
          {({ open }) => (
            <button
              onClick={open}
              disabled={!edited || overwrite.isPending || !validJSON}
              title={!edited ? t("cases.edit_to_enable") : undefined}
              className="inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted disabled:opacity-50"
            >
              {overwrite.isPending ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
              {overwrite.isPending ? t("cases.saving") : t("cases.save_overwrite")}
            </button>
          )}
        </Popconfirm>
        <button
          onClick={saveSnapshot}
          disabled={!edited || snapshot.isPending || !validJSON}
          title={!edited ? t("cases.edit_to_enable") : undefined}
          className="inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted disabled:opacity-50"
        >
          {snapshot.isPending ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
          {snapshot.isPending ? t("cases.saving") : t("cases.save_snapshot")}
        </button>

        <button
          onClick={() => setCurlOpen(true)}
          disabled={!effectiveSession}
          title={!effectiveSession ? t("cases.none_active") : undefined}
          className="inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted disabled:opacity-50"
        >
          <Terminal size={14} /> {t("cases.curl")}
        </button>

        {result && !resultOpen && (
          <button
            onClick={() => setResultOpen(true)}
            className="ml-auto inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
          >
            <PanelRightOpen size={14} /> {t("replay.result")}
          </button>
        )}
      </footer>

      <CurlSheet
        open={curlOpen}
        onOpenChange={setCurlOpen}
        caseId={caseItem.id}
        sessionId={effectiveSession}
        body={edited ? body : undefined}
      />

      <Sheet open={resultOpen} onOpenChange={setResultOpen}>
        <SheetContent width="w-[clamp(420px,38vw,720px)]">
          <SheetA11y title={t("replay.result")} />
          <div className="flex h-full flex-col">
            <header className="border-b px-5 py-3 pr-12">
              <h3 className="text-sm font-semibold">{t("replay.result")}</h3>
              {result && (
                <div className="mt-1 flex flex-wrap items-center gap-x-3 text-xs text-muted-foreground">
                  <span>
                    {t("detail.status")}:{" "}
                    <span className={cn("mono font-medium", result.status >= 400 ? "text-error" : "text-foreground")}>
                      {result.status}
                    </span>
                  </span>
                  <span>{t("detail.duration")}: {result.duration_ms}ms</span>
                  {result.normalized?.response?.usage && (
                    <span className="mono">
                      {t("detail.usage")}: out {result.normalized.response.usage.output_tokens ?? 0}
                    </span>
                  )}
                </div>
              )}
            </header>
            <div className="min-h-0 flex-1 overflow-auto px-5 py-3">
              {run.isError && (
                <p className="mb-2 rounded-md border border-error/40 bg-error/5 px-3 py-2 text-xs text-error">
                  {(run.error as Error).message}
                </p>
              )}
              {result?.normalize_error && <p className="mb-2 text-xs text-error">{result.normalize_error}</p>}
              <ResultBody result={result} />
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

// ResultBody renders the replay outcome: the assistant's final text on success,
// or the raw upstream body on a non-2xx (or when nothing normalized) — so an
// error like 401/insufficient_quota is actually visible instead of a blank "—".
function ResultBody({ result }: { result?: ReplayResult }) {
  const { t } = useTranslation();
  if (!result) return <pre className="whitespace-pre-wrap break-words text-xs leading-relaxed">—</pre>;

  const finalText = result.normalized?.response?.final_text ?? "";
  const isError = result.status >= 400;
  const showRaw = (isError || finalText.trim() === "") && !!result.response_body;

  if (showRaw) {
    return (
      <div>
        {isError && <p className="mb-1 text-xs font-medium text-error">{t("cases.upstream_error")}</p>}
        <pre
          className={cn(
            "whitespace-pre-wrap break-words rounded-md border p-3 text-xs leading-relaxed",
            isError ? "border-error/40 bg-error/5" : "bg-muted/40",
          )}
        >
          {result.response_body}
        </pre>
        {result.truncated && <p className="mt-1 text-[11px] text-muted-foreground">{t("cases.truncated")}</p>}
      </div>
    );
  }
  return (
    <pre className="whitespace-pre-wrap break-words text-xs leading-relaxed">{finalText || "—"}</pre>
  );
}

// CurlSheet builds a copy-pasteable curl for the case + selected session. Proxy
// mode carries no secret; direct mode embeds auth (masked until revealed). The
// curl is fetched server-side so the token/real credentials never enter frontend
// state.
function CurlSheet({
  open,
  onOpenChange,
  caseId,
  sessionId,
  body,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  caseId: string;
  sessionId: string;
  body?: string;
}) {
  const { t } = useTranslation();
  const curl = useCaseCurl(caseId);
  const [mode, setMode] = useState<CurlMode>("proxy");
  const [reveal, setReveal] = useState(false);
  const [copied, setCopied] = useState(false);

  const { mutate } = curl;
  useEffect(() => {
    if (open && sessionId) mutate({ sessionId, mode, reveal, body });
  }, [open, sessionId, mode, reveal, body, mutate]);
  useEffect(() => setCopied(false), [curl.data?.curl]);

  const data = curl.data;
  const switchMode = (m: CurlMode) => {
    setMode(m);
    if (m === "proxy") setReveal(false); // direct always starts masked
  };
  const copy = () => {
    if (!data?.curl) return;
    navigator.clipboard.writeText(data.curl).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="w-[clamp(420px,42vw,820px)]">
        <SheetA11y title={t("cases.curl_title")} />
        <div className="flex h-full flex-col">
          <header className="space-y-3 border-b px-5 py-3 pr-12">
            <h3 className="text-sm font-semibold">{t("cases.curl_title")}</h3>
            <div className="inline-flex rounded-md border p-0.5 text-xs">
              {(["proxy", "direct"] as CurlMode[]).map((m) => (
                <button
                  key={m}
                  type="button"
                  onClick={() => switchMode(m)}
                  className={cn(
                    "rounded px-2.5 py-1 font-medium transition-colors",
                    mode === m ? "bg-accent text-white" : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  {t(`cases.curl_mode_${m}`)}
                </button>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              {mode === "proxy" ? t("cases.curl_proxy_note") : t("cases.curl_direct_note")}
            </p>
          </header>

          <div className="min-h-0 flex-1 overflow-auto px-5 py-3">
            {mode === "direct" && (
              <div className="mb-3 space-y-2 rounded-md border border-error/40 bg-error/5 px-3 py-2">
                <p className="flex items-start gap-1.5 text-xs text-error">
                  <AlertTriangle size={13} className="mt-0.5 shrink-0" />
                  {t("cases.curl_direct_warning")}
                </p>
                {data?.credential_kind === "subscription" && (
                  <p className="text-xs text-error/90">{t("cases.curl_subscription_note")}</p>
                )}
                <button
                  type="button"
                  onClick={() => setReveal((v) => !v)}
                  className="inline-flex items-center gap-1.5 rounded-md border border-error/40 bg-card px-2 py-1 text-xs text-error hover:bg-error/10"
                >
                  {reveal ? <EyeOff size={13} /> : <Eye size={13} />}
                  {reveal ? t("cases.curl_hide") : t("cases.curl_reveal")}
                </button>
              </div>
            )}

            <div className="relative">
              <button
                type="button"
                onClick={copy}
                disabled={!data?.curl}
                className="absolute right-2 top-2 inline-flex items-center gap-1 rounded-md border bg-card px-2 py-1 text-[11px] text-muted-foreground hover:bg-muted disabled:opacity-50"
              >
                {copied ? <Check size={12} /> : <Copy size={12} />}
                {copied ? t("launch.copied") : t("launch.copy")}
              </button>
              {curl.isPending && !data ? (
                <p className="p-3 text-xs text-muted-foreground">{t("detail.loading")}</p>
              ) : curl.isError ? (
                <p className="p-3 text-xs text-error">{(curl.error as Error).message}</p>
              ) : (
                <pre className="mono overflow-auto rounded-md border bg-muted/40 p-3 pr-20 text-xs leading-relaxed">
                  {data?.curl ?? ""}
                </pre>
              )}
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

// ReenterKeyBanner prompts for the API key of a key-mode session that lost it on a
// tracelab restart. The key goes straight to server memory (never disk); the still-
// running agent resumes on its next request, so no relaunch is needed.
function ReenterKeyBanner({ session }: { session: ActiveSession }) {
  const { t } = useTranslation();
  const reenter = useReenterSessionKey();
  const [key, setKey] = useState("");
  const submit = () => {
    const v = key.trim();
    if (!v) return;
    reenter.mutate({ id: session.id, apiKey: v }, { onSuccess: () => setKey("") });
  };
  return (
    <div className="mx-6 mt-3 space-y-2 rounded-md border border-warning/40 bg-warning/5 px-3 py-2">
      <p className="flex items-start gap-1.5 text-xs text-warning">
        <AlertTriangle size={13} className="mt-0.5 shrink-0" />
        {t("cases.needs_key_banner")}
      </p>
      <div className="flex items-center gap-2">
        <input
          type="password"
          value={key}
          onChange={(e) => setKey(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          placeholder={t("cases.needs_key_placeholder")}
          className="mono min-w-0 flex-1 rounded-md border bg-card px-2 py-1 text-xs outline-none focus:border-accent"
        />
        <button
          type="button"
          onClick={submit}
          disabled={!key.trim() || reenter.isPending}
          className="shrink-0 rounded-md bg-accent px-3 py-1 text-xs font-medium text-accent-foreground hover:opacity-90 disabled:opacity-50"
        >
          {reenter.isPending ? t("cases.needs_key_saving") : t("cases.needs_key_save")}
        </button>
      </div>
      {reenter.isError && <p className="text-xs text-error">{(reenter.error as Error).message}</p>}
    </div>
  );
}

// AuthRow renders one auth target compactly: client glyph, upstream, and the
// credential kind (subscription vs API key) — the thing that actually
// distinguishes same-client sessions.
function AuthRow({ s }: { s: ActiveSession }) {
  const { t } = useTranslation();
  return (
    <span className="flex min-w-0 flex-1 items-center gap-2">
      <ClientIcon client={s.client} size={14} />
      <span className="mono min-w-0 flex-1 truncate text-xs text-muted-foreground">{s.upstream}</span>
      {s.needs_key ? (
        <span
          title={t("cases.needs_key_hint")}
          className="inline-flex shrink-0 items-center gap-1 rounded bg-warning/15 px-1.5 py-0.5 text-[10px] font-medium text-warning"
        >
          <AlertTriangle size={10} />
          {t("cases.needs_key")}
        </span>
      ) : (
        <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
          {t(`cases.cred_${s.credential_kind}`)}
        </span>
      )}
    </span>
  );
}

function CloseAuthButton({ s, onClosed }: { s: ActiveSession; onClosed: (id: string) => void }) {
  const { t } = useTranslation();
  return (
    <Popconfirm
      message={t("cases.close_session_confirm")}
      confirmLabel={t("cases.close_session")}
      cancelLabel={t("replay.cancel")}
      tone="danger"
      placement="bottom"
      onConfirm={() => onClosed(s.id)}
    >
      {({ open }) => (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            open();
          }}
          title={t("cases.close_session")}
          className="mr-1 shrink-0 rounded p-1 text-muted-foreground hover:bg-card hover:text-error"
        >
          <X size={14} />
        </button>
      )}
    </Popconfirm>
  );
}

// AuthSelect picks the auth (a live session) a case runs against. Sessions are
// already filtered to the case's provider and deduped to distinct auth targets,
// so a single target needs no chooser (just a label). The dropdown is rendered
// INLINE (no body portal) so it stays clickable inside the runner's modal dialog.
function AuthSelect({
  targets,
  value,
  label,
  onChange,
}: {
  targets: ActiveSession[];
  value: string;
  label: string;
  onChange: (id: string) => void;
}) {
  const close = useCloseActiveSession();
  const [open, setOpen] = useState(false);
  const selected = targets.find((s) => s.id === value) ?? targets[0];
  if (!selected) return null;
  const multi = targets.length > 1;

  const doClose = (id: string) => {
    close.mutate(id);
    if (id === value) onChange(""); // drop a dead selection; falls back to first remaining
    setOpen(false);
  };

  if (!multi) {
    return (
      <div className="flex items-center gap-1 rounded-md border bg-card px-3 py-2" title={label}>
        <AuthRow s={selected} />
        <CloseAuthButton s={selected} onClosed={doClose} />
      </div>
    );
  }

  return (
    <div className="relative">
      <button
        type="button"
        title={label}
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 rounded-md border bg-card px-3 py-2 text-left shadow-sm transition-colors hover:bg-muted/50"
      >
        <AuthRow s={selected} />
        <ChevronDown size={16} className="shrink-0 text-muted-foreground" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 z-20 mt-1 max-h-80 w-[min(440px,80vw)] overflow-auto rounded-lg border bg-card p-1.5 shadow-lg">
            {targets.map((s) => (
              <div
                key={s.id}
                className={cn(
                  "group flex items-center gap-1 rounded-md hover:bg-muted",
                  s.id === selected.id && "bg-accent/8",
                )}
              >
                <button
                  type="button"
                  onClick={() => {
                    onChange(s.id);
                    setOpen(false);
                  }}
                  className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-2 text-left"
                >
                  <Check size={15} className={cn("shrink-0 text-accent", s.id !== selected.id && "opacity-0")} />
                  <AuthRow s={s} />
                </button>
                <CloseAuthButton s={s} onClosed={doClose} />
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function CaseTags({ tags }: { tags: string }) {
  const tokens = caseTagTokens(tags);
  if (tokens.length === 0) return null;
  return (
    <div className="flex min-w-0 flex-wrap gap-1.5 pt-0.5">
      {tokens.map((tag) => (
        <span
          key={tag}
          className="inline-flex max-w-full items-center rounded-md border bg-muted/60 px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground"
        >
          <span className="text-accent">#</span>
          <span className="ml-0.5 truncate">{tag}</span>
        </span>
      ))}
    </div>
  );
}

function caseTagTokens(tags: string): string[] {
  return tags
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
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
