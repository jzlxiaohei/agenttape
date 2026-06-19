import { lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { ArrowUpRight } from "lucide-react";
import type { ContentBlock } from "@/api/events";
import { useUIStore } from "@/store/ui";
import { useDetailLinks } from "./DetailLinks";
import { Expandable } from "./Expandable";

const Markdown = lazy(() => import("./Markdown"));

const LONG_TEXT = 800;
const LONG_JSON = 600;

// Renders typed content blocks. Types come from the normalized structure, not
// keyword guessing — so each kind gets its own faithful rendering. Long content
// is capped with a show-more toggle.
export function ContentBlocks({ blocks }: { blocks?: ContentBlock[] }) {
  if (!blocks || blocks.length === 0) return null;
  return (
    <div className="space-y-2">
      {blocks.map((b, i) => (
        <BlockView key={i} block={b} />
      ))}
    </div>
  );
}

function BlockView({ block }: { block: ContentBlock }) {
  const { t } = useTranslation();
  switch (block.type) {
    case "text":
      return <Text text={block.text ?? ""} />;
    case "reasoning":
      return (
        <div className="rounded-lg border-l-2 border-reasoning bg-reasoning/5 px-3 py-2">
          <div className="mb-1 text-xs font-medium text-reasoning">{t("block.reasoning")}</div>
          <Text text={block.text ?? ""} muted />
        </div>
      );
    case "tool_call":
      return <ToolCallBlock block={block} />;
    case "tool_result":
      return (
        <div className="rounded-lg border border-toolresult/30 bg-toolresult/5 px-3 py-2">
          <div className="mb-1 text-xs font-medium text-toolresult">
            {t("block.tool_result")}
            {block.tool_result?.is_error && <span className="ml-1 text-error">⚠</span>}
          </div>
          <ContentBlocks blocks={block.tool_result?.content} />
        </div>
      );
    case "image":
      return <div className="text-xs italic text-muted-foreground">[{t("block.image")}]</div>;
    default:
      return <div className="text-xs italic text-muted-foreground">[{t("block.unknown")}]</div>;
  }
}

// A tool_call block, with a jump to its harness hook (PreToolUse) if one exists
// in the session — connecting the request layer to the orchestration layer.
function ToolCallBlock({ block }: { block: ContentBlock }) {
  const { t } = useTranslation();
  const links = useDetailLinks();
  const selectEvent = useUIStore((s) => s.selectEvent);
  const hook = links.hookForToolCall(block.tool_call?.id);
  return (
    <div className="rounded-lg border border-toolcall/30 bg-toolcall/5 px-3 py-2">
      <div className="mb-1 flex items-center justify-between gap-2 text-xs font-medium text-toolcall">
        <span>
          {t("block.tool_call")}: <span className="mono">{block.tool_call?.name}</span>
        </span>
        {hook && (
          <button
            onClick={() => selectEvent(hook.id)}
            className="inline-flex items-center gap-1 rounded-md border border-accent/40 px-1.5 py-0.5 text-accent hover:bg-accent/10"
          >
            <ArrowUpRight size={12} />
            {t("link.to_hook")}
          </button>
        )}
      </div>
      <JsonPreview value={block.tool_call?.arguments} />
    </div>
  );
}

function Text({ text, muted }: { text: string; muted?: boolean }) {
  const renderMarkdown = useUIStore((s) => s.renderMarkdown);
  const cls = `whitespace-pre-wrap break-words text-sm leading-relaxed${muted ? " text-muted-foreground" : ""}`;
  const raw = <p className={cls}>{text}</p>;
  // Default raw (fidelity). When rendering, lazy-load markdown; fall back to raw
  // text while the chunk loads so there is no flash of nothing.
  const node = renderMarkdown ? (
    <Suspense fallback={raw}>
      <div className={muted ? "text-muted-foreground" : undefined}>
        <Markdown text={text} />
      </div>
    </Suspense>
  ) : (
    raw
  );
  return text.length > LONG_TEXT ? <Expandable>{node}</Expandable> : node;
}

export function JsonPreview({ value }: { value: unknown }) {
  if (value == null) return null;
  const text = typeof value === "string" ? value : JSON.stringify(value, null, 2);
  const node = (
    <pre className="mono overflow-x-auto whitespace-pre-wrap break-words rounded bg-muted/60 p-2 text-xs">
      {text}
    </pre>
  );
  return text.length > LONG_JSON ? <Expandable maxHeight={200}>{node}</Expandable> : node;
}
