import { useState } from "react";
import { useTranslation } from "react-i18next";

// Caps tall content and reveals a show-more/less toggle. The expanded flag is
// pure view-local state (nobody else reads it), so useState is correct here.
export function Expandable({ maxHeight = 320, children }: { maxHeight?: number; children: React.ReactNode }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  return (
    <div>
      <div
        className="relative overflow-hidden"
        style={{ maxHeight: expanded ? undefined : maxHeight }}
      >
        {children}
        {!expanded && (
          <div className="pointer-events-none absolute inset-x-0 bottom-0 h-8 bg-gradient-to-t from-card to-transparent" />
        )}
      </div>
      <button
        onClick={() => setExpanded((v) => !v)}
        className="mt-1 text-xs text-accent hover:underline"
      >
        {expanded ? t("detail.collapse") : t("detail.expand")}
      </button>
    </div>
  );
}
