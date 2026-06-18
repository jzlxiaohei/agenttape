import { cn } from "@/lib/utils";

export interface TabItem {
  key: string;
  label: string;
  count?: number;
}

// Minimal underline tab bar. Pure presentation.
export function Tabs({
  items,
  active,
  onChange,
}: {
  items: TabItem[];
  active: string;
  onChange: (key: string) => void;
}) {
  return (
    <div className="flex gap-1 border-b">
      {items.map((it) => (
        <button
          key={it.key}
          onClick={() => onChange(it.key)}
          className={cn(
            "-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors",
            active === it.key
              ? "border-accent text-accent"
              : "border-transparent text-muted-foreground hover:text-foreground",
          )}
        >
          {it.label}
          {it.count != null && <span className="ml-1.5 text-xs text-muted-foreground">{it.count}</span>}
        </button>
      ))}
    </div>
  );
}
