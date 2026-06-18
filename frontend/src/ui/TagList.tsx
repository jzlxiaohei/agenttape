import { useTranslation } from "react-i18next";
import type { TagInfo } from "@/api/events";
import { cn } from "@/lib/utils";

// Tags. Structural tags are facts; suspected tags are visually marked uncertain
// (dashed + italic) with an evidence tooltip — honesty first (next.md 3.3).
export function TagList({ tags }: { tags: TagInfo[] }) {
  const { t } = useTranslation();
  if (tags.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-1.5">
      {tags.map((tag, i) => (
        <span
          key={i}
          title={tag.evidence || undefined}
          className={cn(
            "inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium",
            tag.suspected
              ? "border border-dashed border-suspected/60 italic text-suspected"
              : "bg-accent/12 text-accent",
          )}
        >
          {t(`tag.${tag.tag}`, { defaultValue: tag.tag })}
          {tag.suspected && <span className="ml-1">{t("tag.suspected")}</span>}
        </span>
      ))}
    </div>
  );
}
