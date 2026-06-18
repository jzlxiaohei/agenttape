import { useTranslation } from "react-i18next";
import type { Round } from "@/viewmodel/detail";
import { Collapsible } from "./Collapsible";
import { MessageThread } from "./MessageThread";

// Renders request messages grouped into rounds (one user turn + its follow-up
// reasoning/tool steps). Each round is collapsed by default so the whole request
// reads as a navigable overview first.
export function RoundsView({ rounds }: { rounds: Round[] }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-1">
      {rounds.map((r) => {
        const subtitleParts = [
          r.preview,
          r.toolCalls > 0 ? t("round.tools", { count: r.toolCalls }) : "",
        ].filter(Boolean);
        return (
          <Collapsible
            key={r.key}
            sectionKey={r.key}
            title={r.index === 0 ? t("round.preamble") : t("round.n", { n: r.index })}
            subtitle={subtitleParts.join("  ·  ")}
            defaultCollapsed
          >
            <MessageThread messages={r.messages} markSteps />
          </Collapsible>
        );
      })}
    </div>
  );
}
