import { useTranslation } from "react-i18next";
import { Search as SearchIcon } from "lucide-react";
import { useUIStore } from "@/store/ui";
import { useSessionRoute } from "@/viewmodel/route";
import { useSearch, useFacets } from "@/query/search";
import type { SearchResult } from "@/api/search";
import { cn } from "@/lib/utils";
import { Select } from "@/components/ui/select";
import { ClientIcon } from "./ClientIcon";

// Cross-session search: full-text query + tag/provider/client facets. Selecting
// a result opens that event in the Sessions detail view.
export function SearchView() {
  const { t } = useTranslation();
  const q = useUIStore((s) => s.searchQuery);
  const tag = useUIStore((s) => s.searchTag);
  const provider = useUIStore((s) => s.searchProvider);
  const client = useUIStore((s) => s.searchClient);
  const setQuery = useUIStore((s) => s.setSearchQuery);
  const setFilter = useUIStore((s) => s.setSearchFilter);

  const { data: facets } = useFacets();
  const { data: results, isFetching } = useSearch({ q, tag, provider, client });

  return (
    <div className="mx-auto flex h-full w-full max-w-3xl flex-col gap-3 p-6">
      <div className="flex items-center gap-2 rounded-lg border bg-card px-3 py-2">
        <SearchIcon size={16} className="text-muted-foreground" />
        <input
          value={q}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("search.placeholder")}
          className="w-full bg-transparent text-sm outline-none"
        />
      </div>

      <div className="flex flex-wrap gap-2 text-xs">
        <FacetSelect label={t("search.client")} value={client} options={facets?.clients} onChange={(v) => setFilter("searchClient", v)} render={(c) => t(`client.${c}`, { defaultValue: c })} />
        <FacetSelect label={t("search.provider")} value={provider} options={facets?.providers} onChange={(v) => setFilter("searchProvider", v)} />
        <FacetSelect label={t("search.tag")} value={tag} options={facets?.tags} onChange={(v) => setFilter("searchTag", v)} render={(x) => t(`tag.${x}`, { defaultValue: x })} />
        <span className="ml-auto self-center text-muted-foreground">
          {isFetching ? t("search.searching") : t("search.count", { count: results?.length ?? 0 })}
        </span>
      </div>

      <div className="min-h-0 flex-1 space-y-1 overflow-y-auto">
        {(results ?? []).map((r) => (
          <ResultRow key={r.event_id} result={r} />
        ))}
      </div>
    </div>
  );
}

function FacetSelect({
  label,
  value,
  options,
  onChange,
  render,
}: {
  label: string;
  value: string;
  options?: string[];
  onChange: (v: string) => void;
  render?: (v: string) => string;
}) {
  const selectOptions = [
    { value: "", label },
    ...(options ?? []).map((o) => ({ value: o, label: render ? render(o) : o })),
  ];

  return (
    <Select
      value={value}
      onChange={onChange}
      options={selectOptions}
      className="w-36"
      buttonClassName={cn("min-h-7 px-2 py-1 text-xs", value && "border-accent/50 text-accent")}
      optionClassName="text-xs"
    />
  );
}

function ResultRow({ result }: { result: SearchResult }) {
  const { t } = useTranslation();
  const openEvent = useSessionRoute().openEvent;
  const go = () => openEvent(result.session_id, result.event_id);
  return (
    <button
      onClick={go}
      className="block w-full rounded-lg border bg-card px-3 py-2 text-left transition-colors hover:bg-muted/50"
    >
      <div className="flex items-center justify-between gap-2 text-xs">
        <span className="flex items-center gap-1.5 font-medium">
          <ClientIcon client={result.client} size={13} />
          {t(`client.${result.client}`, { defaultValue: result.client })}
          {result.model && <span className="ml-1 text-muted-foreground">{result.model}</span>}
        </span>
        <span className="shrink-0 text-muted-foreground mono">{fmt(result.started_at)}</span>
      </div>
      {result.snippet && (
        <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{result.snippet}</p>
      )}
    </button>
  );
}

function fmt(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleString();
}
