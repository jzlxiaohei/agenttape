import { useTranslation } from "react-i18next";
import { useEventDetail } from "@/query/events";
import { diffMessages, requestSequence, summarizeDiff, type DiffOp } from "@/viewmodel/diff";
import { ContentBlocks } from "./ContentBlocks";
import { Collapsible } from "./Collapsible";
import { cn } from "@/lib/utils";

// Message-level diff of two completions' request input: what the harness added
// or removed turn-to-turn. A large removed run replaced by a summary is the
// visual signature of compaction (next.md 5.1).
export function SemanticDiff({ leftId, rightId }: { leftId: string; rightId: string }) {
  const { t } = useTranslation();
  const left = useEventDetail(leftId);
  const right = useEventDetail(rightId);

  if (left.isLoading || right.isLoading)
    return <p className="text-xs text-muted-foreground">{t("raw.loading")}</p>;

  const ops = diffMessages(requestSequence(left.data), requestSequence(right.data));
  const added = ops.filter((o) => o.kind === "added").length;
  const removed = ops.filter((o) => o.kind === "removed").length;
  const sum = summarizeDiff(ops);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <span className="font-medium text-toolcall">+{added} {t("diff.added")}</span>
        <span className="font-medium text-error">-{removed} {t("diff.removed")}</span>
        {sum.toolResultsAdded > 0 && (
          <span className="rounded-md bg-toolresult/10 px-2 py-0.5 text-toolresult">
            {t("diff.tool_results", { count: sum.toolResultsAdded })}
          </span>
        )}
        {sum.systemChanged && (
          <span className="rounded-md bg-accent/10 px-2 py-0.5 text-accent">{t("diff.system_changed")}</span>
        )}
        {sum.compactionSuspected && (
          <span className="rounded-md border border-dashed border-suspected/50 px-2 py-0.5 italic text-suspected">
            {t("diff.compaction_suspected")}
          </span>
        )}
      </div>
      <div className="space-y-2">
        {groupOps(ops).map((g) =>
          g.kind === "run" ? (
            <Collapsible
              key={g.key}
              sectionKey={`diff-unchanged-${g.key}`}
              title={t("diff.unchanged", { count: g.ops.length })}
              defaultCollapsed
            >
              <div className="space-y-2 opacity-70">
                {g.ops.map((o, i) => (
                  <DiffRow key={i} op={o} />
                ))}
              </div>
            </Collapsible>
          ) : (
            <DiffRow key={g.key} op={g.op} />
          ),
        )}
      </div>
    </div>
  );
}

type Group = { kind: "run"; ops: DiffOp[]; key: string } | { kind: "op"; op: DiffOp; key: string };

// Pure: collapses consecutive unchanged ops into runs so the UI can fold them.
function groupOps(ops: DiffOp[]): Group[] {
  const groups: Group[] = [];
  let run: DiffOp[] = [];
  let runStart = 0;
  const flush = () => {
    if (run.length) {
      groups.push({ kind: "run", ops: run, key: `${runStart}` });
      run = [];
    }
  };
  ops.forEach((op, idx) => {
    if (op.kind === "unchanged") {
      if (run.length === 0) runStart = idx;
      run.push(op);
    } else {
      flush();
      groups.push({ kind: "op", op, key: `${idx}` });
    }
  });
  flush();
  return groups;
}

const style: Record<string, { wrap: string; mark: string; sign: string }> = {
  added: { wrap: "border-l-2 border-toolcall bg-toolcall/5", mark: "text-toolcall", sign: "+" },
  removed: { wrap: "border-l-2 border-error bg-error/5 opacity-80", mark: "text-error", sign: "−" },
  unchanged: { wrap: "border-l-2 border-border", mark: "text-muted-foreground", sign: "=" },
};

function DiffRow({ op }: { op: DiffOp }) {
  const { t } = useTranslation();
  const s = style[op.kind];
  const role = t(`role.${op.message.role}`, { defaultValue: op.message.role });
  return (
    <div className={cn("rounded-r-lg px-3 py-2", s.wrap)}>
      <div className={cn("mb-1 flex items-center gap-2 text-xs font-semibold uppercase", s.mark)}>
        <span>{s.sign}</span>
        <span>{role}</span>
      </div>
      <ContentBlocks blocks={op.message.content} />
    </div>
  );
}
