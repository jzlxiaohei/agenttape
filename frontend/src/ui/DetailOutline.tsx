export interface OutlineItem {
  key: string;
  label: string;
  count?: number;
}

// Structure-grouped navigation (not per-message): jump to System / Tools /
// Messages / Response. Clicking scrolls the matching section into view.
export function DetailOutline({ items }: { items: OutlineItem[] }) {
  if (items.length === 0) return null;
  const go = (key: string) =>
    document.getElementById(`section-${key}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
  return (
    <nav className="sticky top-0 space-y-1 py-2 text-xs">
      {items.map((it) => (
        <button
          key={it.key}
          onClick={() => go(it.key)}
          className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1 text-left text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <span className="truncate">{it.label}</span>
          {it.count != null && <span className="shrink-0">{it.count}</span>}
        </button>
      ))}
    </nav>
  );
}
