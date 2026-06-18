import { useTranslation } from "react-i18next";
import type { SectionBar } from "@/viewmodel/detail";

// Section token share as a stacked CSS bar + legend. Pure presentation.
export function TokenBars({ bars }: { bars: SectionBar[] }) {
  const { t } = useTranslation();
  if (bars.length === 0) return null;
  return (
    <div className="space-y-2">
      <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted">
        {bars.map((b) => (
          <div key={b.name} style={{ width: `${b.pct}%`, background: b.color }} title={`${b.name} ${b.pct.toFixed(1)}%`} />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {bars.map((b) => (
          <span key={b.name} className="inline-flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-sm" style={{ background: b.color }} />
            {t(`section.${b.name}`, { defaultValue: b.name })}
            <span className="text-foreground">{b.pct.toFixed(1)}%</span>
            <span className="mono">({b.tokens})</span>
          </span>
        ))}
      </div>
    </div>
  );
}
