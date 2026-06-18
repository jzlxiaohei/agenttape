import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Tool } from "@/api/events";
import { JsonPreview } from "./ContentBlocks";

// Tool definitions sent in the request. Studying these matters — tools often
// dominate prompt tokens. Schema is collapsed by default per tool.
export function ToolsView({ tools }: { tools: Tool[] }) {
  return (
    <div className="space-y-2">
      {tools.map((tool, i) => (
        <ToolRow key={i} tool={tool} />
      ))}
    </div>
  );
}

function ToolRow({ tool }: { tool: Tool }) {
  const { t } = useTranslation();
  const [showSchema, setShowSchema] = useState(false);
  return (
    <div className="rounded-lg border bg-card px-3 py-2">
      <div className="flex items-center justify-between gap-2">
        <span className="mono text-sm font-medium text-toolcall">{tool.name}</span>
        {tool.input_schema != null && (
          <button
            onClick={() => setShowSchema((v) => !v)}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            {showSchema ? t("tools.hide_schema") : t("tools.show_schema")}
          </button>
        )}
      </div>
      {tool.description && (
        <p className="mt-1 whitespace-pre-wrap break-words text-xs text-muted-foreground">
          {tool.description}
        </p>
      )}
      {showSchema && (
        <div className="mt-2">
          <JsonPreview value={tool.input_schema} />
        </div>
      )}
    </div>
  );
}
