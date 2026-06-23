import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { foldAll, unfoldAll, forceParsing } from "@codemirror/language";
import type { EditorView } from "@codemirror/view";
import { useRawFile } from "@/query/events";
import { cn } from "@/lib/utils";
import { CopyButton } from "./CopyButton";

type Role = "request_body" | "response_body";

// Escape-hatch raw viewer: exact captured bytes in CodeMirror. JSON is
// pretty-printed and foldable by field (fold gutter + fold all/unfold all);
// non-JSON (SSE) is shown verbatim. Download grabs the original bytes. This is
// the foundation for the future jq-style explore + diff.
export function RawView({ eventId }: { eventId: string }) {
  const { t } = useTranslation();
  const [role, setRole] = useState<Role | null>("request_body");
  const { data, isLoading } = useRawFile(eventId, role ?? "", role !== null);
  const ref = useRef<ReactCodeMirrorRef>(null);

  const raw = data ?? "";
  const pretty = prettyJSON(raw);

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        {(["request_body", "response_body"] as const).map((r) => (
          <button
            key={r}
            onClick={() => setRole(role === r ? null : r)}
            className={cn(
              "rounded-md border px-2 py-1 text-xs",
              role === r ? "border-accent/60 bg-accent/10 text-accent" : "text-muted-foreground hover:bg-muted",
            )}
          >
            {t(`raw.${r}`)}
          </button>
        ))}
        {role && !isLoading && (
          <div className="ml-auto flex items-center gap-2 text-xs">
            <button className="text-muted-foreground hover:text-foreground" onClick={() => ref.current?.view && foldAll(ref.current.view)}>
              {t("raw.fold_all")}
            </button>
            <button className="text-muted-foreground hover:text-foreground" onClick={() => ref.current?.view && unfoldAll(ref.current.view)}>
              {t("raw.unfold_all")}
            </button>
            <CopyButton text={pretty} />
            <button className="text-accent hover:underline" onClick={() => download(`${eventId}.${role}.txt`, raw)}>
              {t("raw.download")}
            </button>
          </div>
        )}
      </div>
      {role && (
        <div className="overflow-hidden rounded-lg border">
          {isLoading ? (
            <p className="p-3 text-xs text-muted-foreground">{t("raw.loading")}</p>
          ) : (
            <CodeMirror
              ref={ref}
              value={pretty}
              height="420px"
              editable={false}
              extensions={[json()]}
              basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: false }}
              onCreateEditor={parseWholeDoc}
            />
          )}
        </div>
      )}
    </div>
  );
}

// Large docs are parsed incrementally, so fold markers near the top only appear
// after scrolling. Force a full parse on mount so every field is foldable
// immediately.
function parseWholeDoc(view: EditorView) {
  forceParsing(view, view.state.doc.length, 5000);
}

function prettyJSON(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text; // SSE / non-JSON stays verbatim
  }
}

function download(filename: string, text: string) {
  const url = URL.createObjectURL(new Blob([text], { type: "text/plain" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
