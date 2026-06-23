import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { ArrowRight, Beaker } from "lucide-react";
import type { Experiment } from "@/lib/experiments";
import { CopyButton } from "./CopyButton";

// A non-runnable instruction card for a hands-on experiment: a "needs hooks"
// note, a short why, then concise numbered steps (core steps annotated, one
// carries a copy-pasteable prompt), and a link to the Launch page.
export function ExperimentCard({ experiment }: { experiment: Experiment }) {
  const { t } = useTranslation();
  const base = `experiments.${experiment.id}`;

  return (
    <div className="rounded-lg border border-reasoning/30 bg-reasoning/[0.03] p-4">
      <div className="flex items-center gap-2">
        <Beaker size={15} className="text-reasoning" />
        <h3 className="text-sm font-semibold">{t(`${base}.title`)}</h3>
        <span className="rounded bg-reasoning/15 px-1.5 py-0.5 text-[10px] font-medium text-reasoning">
          {t("experiments.needs_hooks")}
        </span>
      </div>
      <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">{t(`${base}.desc`)}</p>

      <ol className="mt-3 space-y-3">
        {experiment.steps.map((s, i) => {
          const n = i + 1;
          const sb = `${base}.s${n}`;
          return (
            <li key={n} className="flex gap-2.5">
              <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-reasoning/15 text-[11px] font-semibold text-reasoning">
                {n}
              </span>
              <div className="min-w-0 flex-1 space-y-1">
                <p className="text-sm leading-snug">{t(`${sb}.label`)}</p>
                {s.note && (
                  <p className="text-xs italic leading-relaxed text-muted-foreground">{t(`${sb}.note`)}</p>
                )}
                {s.prompt && (
                  <div className="relative mt-1">
                    <div className="absolute right-1.5 top-1.5 z-10">
                      <CopyButton text={t(`${sb}.prompt`)} />
                    </div>
                    <pre className="whitespace-pre-wrap break-words rounded-md border bg-card p-2.5 pr-16 text-xs leading-relaxed">
                      {t(`${sb}.prompt`)}
                    </pre>
                  </div>
                )}
              </div>
            </li>
          );
        })}
      </ol>

      <Link
        to="/launch"
        className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-accent hover:underline"
      >
        {t("experiments.open_launch")} <ArrowRight size={13} />
      </Link>
    </div>
  );
}
