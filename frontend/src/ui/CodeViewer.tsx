import { useRef } from "react";
import { useTranslation } from "react-i18next";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { foldAll, unfoldAll, forceParsing } from "@codemirror/language";
import type { EditorView } from "@codemirror/view";

// Default export so it can be lazy-loaded (CodeMirror is heavy). Read-only JSON
// viewer: pretty-printed, foldable by field, downloadable. Shared by the hook
// payload view and any other raw-JSON display.
export default function CodeViewer({
  text,
  filename,
  height = "420px",
}: {
  text: string;
  filename: string;
  height?: string;
}) {
  const { t } = useTranslation();
  const ref = useRef<ReactCodeMirrorRef>(null);
  const pretty = prettyJSON(text);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 text-xs">
        <button className="text-muted-foreground hover:text-foreground" onClick={() => ref.current?.view && foldAll(ref.current.view)}>
          {t("raw.fold_all")}
        </button>
        <button className="text-muted-foreground hover:text-foreground" onClick={() => ref.current?.view && unfoldAll(ref.current.view)}>
          {t("raw.unfold_all")}
        </button>
        <button className="text-accent hover:underline" onClick={() => download(filename, text)}>
          {t("raw.download")}
        </button>
      </div>
      <div className="overflow-hidden rounded-lg border">
        <CodeMirror
          ref={ref}
          value={pretty}
          height={height}
          editable={false}
          extensions={[json()]}
          basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: false }}
          onCreateEditor={(v: EditorView) => forceParsing(v, v.state.doc.length, 5000)}
        />
      </div>
    </div>
  );
}

function prettyJSON(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
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
