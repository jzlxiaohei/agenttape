import { ChevronRight } from "lucide-react";
import { useUIStore } from "@/store/ui";
import { cn } from "@/lib/utils";

// A section whose collapsed state lives in the store (so the outline and filters
// can interact with it). defaultCollapsed applies until the user toggles.
export function Collapsible({
  sectionKey,
  title,
  subtitle,
  count,
  accent,
  defaultCollapsed = false,
  children,
}: {
  sectionKey: string;
  title: string;
  subtitle?: string;
  count?: number;
  accent?: string;
  defaultCollapsed?: boolean;
  children: React.ReactNode;
}) {
  const override = useUIStore((s) => s.collapsed[sectionKey]);
  const toggle = useUIStore((s) => s.toggleCollapsed);
  const collapsed = override ?? defaultCollapsed;

  return (
    <section id={`section-${sectionKey}`} className="scroll-mt-4">
      <button
        onClick={() => toggle(sectionKey)}
        className="flex w-full items-center gap-2 py-2 text-left"
      >
        <ChevronRight
          size={16}
          className={cn("shrink-0 text-muted-foreground transition-transform", !collapsed && "rotate-90")}
        />
        <span className="shrink-0 text-sm font-semibold" style={accent ? { color: accent } : undefined}>
          {title}
        </span>
        {count != null && <span className="shrink-0 text-xs text-muted-foreground">{count}</span>}
        {subtitle && (
          <span className="truncate text-xs font-normal text-muted-foreground">{subtitle}</span>
        )}
      </button>
      {!collapsed && <div className="pb-4 pl-6">{children}</div>}
    </section>
  );
}
