import { useTranslation } from "react-i18next";
import { Minimize2 } from "lucide-react";
import type { CompactionView } from "@/viewmodel/detail";
import type { CompactionGrade } from "@/api/events";
import { cn } from "@/lib/utils";
import { Collapsible } from "./Collapsible";
import { CopyButton } from "./CopyButton";

// Graded badge styling: confirmed = solid evidence (hook), strong = lineage
// proven, weak = shrink only (low-saturation warning per design's "suspected").
const gradeStyle: Record<CompactionGrade, string> = {
  confirmed: "bg-accent/15 text-accent",
  strong_suspected: "bg-reasoning/15 text-reasoning",
  weak_suspected: "bg-muted text-muted-foreground",
};

// Option-A compaction view: the req↔res comparison for a /compact request — how
// much context went in (incl. cache) vs the summary that came out, plus the
// summary text itself. The grade comes from the cross-event episode detector.
export function CompactionPanel({
  data,
  grade,
  evidence,
}: {
  data: CompactionView;
  grade: CompactionGrade;
  evidence?: string;
}) {
  const { t } = useTranslation();
  const { contextIn, summaryOut, summaryText } = data;
  const pct = contextIn > 0 ? (summaryOut / contextIn) * 100 : 0;
  const fmt = (n: number) => n.toLocaleString();

  return (
    <div className="rounded-lg border border-reasoning/40 bg-reasoning/5 p-4">
      <div className="flex items-center gap-2">
        <Minimize2 size={15} className="text-reasoning" />
        <h3 className="text-sm font-semibold">{t("compaction.title")}</h3>
        <span
          className={cn("rounded px-1.5 py-0.5 text-[10px] font-medium", gradeStyle[grade])}
          title={evidence}
        >
          {t(`compaction.grade.${grade}`)}
        </span>
      </div>
      <p className="mt-1 text-xs text-muted-foreground">{t("compaction.hint")}</p>

      <div className="mt-3">
        <div className="mb-1 flex items-center justify-between text-xs">
          <span className="mono text-muted-foreground">
            {t("compaction.context_in")}: <span className="font-medium text-foreground">{fmt(contextIn)}</span>
          </span>
          <span className="mono text-muted-foreground">
            {t("compaction.summary_out")}: <span className="font-medium text-foreground">{fmt(summaryOut)}</span>
          </span>
        </div>
        <div className="h-2 w-full overflow-hidden rounded-full bg-muted" title={`${fmt(summaryOut)} / ${fmt(contextIn)}`}>
          <div className="h-full rounded-full bg-reasoning" style={{ width: `${Math.min(Math.max(pct, 0.5), 100)}%` }} />
        </div>
        <p className="mt-1 text-[11px] text-muted-foreground">{t("compaction.compressed_to", { pct: pct.toFixed(1) })}</p>
      </div>

      {summaryText && (
        <div className="mt-3">
          <Collapsible
            sectionKey="compaction-summary"
            title={t("compaction.summary")}
            accent="var(--color-reasoning)"
            defaultCollapsed
          >
            <div className="relative">
              <div className="absolute right-1 top-1 z-10">
                <CopyButton text={summaryText} />
              </div>
              <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-card p-3 pr-16 text-xs leading-relaxed">
                {summaryText}
              </pre>
            </div>
          </Collapsible>
        </div>
      )}
    </div>
  );
}
