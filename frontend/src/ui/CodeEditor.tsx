import { useRef } from "react";
import { useTranslation } from "react-i18next";
import CodeMirror from "@uiw/react-codemirror";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { foldAll, unfoldAll, forceParsing, foldService } from "@codemirror/language";
import type { EditorState } from "@codemirror/state";
import type { EditorView } from "@codemirror/view";
import { EditorView as EditorViewExtension } from "@codemirror/view";
import { cn } from "@/lib/utils";

// Editable JSON editor (CodeMirror) — line numbers, folding, bracket matching/
// closing. Default export so it loads with its (lazy) host. The read-only sibling
// is CodeViewer. Pass height="100%" + a flex-1 className to let it fill its parent.
export default function CodeEditor({
  value,
  onChange,
  height = "300px",
  className,
  showFoldControls = false,
}: {
  value: string;
  onChange: (v: string) => void;
  height?: string;
  className?: string;
  showFoldControls?: boolean;
}) {
  const { t } = useTranslation();
  const ref = useRef<ReactCodeMirrorRef>(null);
  const fill = height === "100%";
  return (
    <div className={cn("flex w-full min-w-0 max-w-full flex-col overflow-hidden", fill && "h-full", className)}>
      {showFoldControls && (
        <div className="mb-1 flex items-center gap-2 text-xs">
          <button
            className="text-muted-foreground hover:text-foreground"
            onClick={() => ref.current?.view && foldAll(ref.current.view)}
            type="button"
          >
            {t("raw.fold_all")}
          </button>
          <button
            className="text-muted-foreground hover:text-foreground"
            onClick={() => ref.current?.view && unfoldAll(ref.current.view)}
            type="button"
          >
            {t("raw.unfold_all")}
          </button>
        </div>
      )}
      <div className={cn("w-full min-h-0 min-w-0 max-w-full overflow-hidden rounded-lg border", fill && "flex-1")}>
        <CodeMirror
          ref={ref}
          value={value}
          height={height}
          className={cn("tracelab-code-editor w-full max-w-full", fill && "h-full")}
          editable
          extensions={[json(), foldService.of(jsonBracketFold), EditorViewExtension.lineWrapping]}
          basicSetup={{
            lineNumbers: true,
            foldGutter: true,
            highlightActiveLine: true,
            bracketMatching: true,
            closeBrackets: true,
            autocompletion: false,
          }}
          onChange={onChange}
          onCreateEditor={(v: EditorView) => forceParsing(v, v.state.doc.length, 5000)}
        />
      </div>
    </div>
  );
}

function jsonBracketFold(state: EditorState, lineStart: number) {
  const line = state.doc.lineAt(lineStart);
  const opener = lastOpenerOnLine(line.text);
  if (!opener) return null;

  let depth = 1;
  for (let lineNo = line.number + 1; lineNo <= state.doc.lines; lineNo += 1) {
    const cur = state.doc.line(lineNo);
    for (const token of bracketTokens(cur.text)) {
      if (token.char === opener.open) depth += 1;
      if (token.char === opener.close) depth -= 1;
      if (depth === 0) {
        return { from: line.from + opener.index + 1, to: cur.from + token.index };
      }
    }
  }
  return null;
}

function lastOpenerOnLine(text: string): { open: "{" | "["; close: "}" | "]"; index: number } | null {
  let found: { open: "{" | "["; close: "}" | "]"; index: number } | null = null;
  for (const token of bracketTokens(text)) {
    if (token.char === "{") found = { open: "{", close: "}", index: token.index };
    if (token.char === "[") found = { open: "[", close: "]", index: token.index };
    if (token.char === "}" || token.char === "]") found = null;
  }
  return found;
}

function bracketTokens(text: string): Array<{ char: "{" | "}" | "[" | "]"; index: number }> {
  const out: Array<{ char: "{" | "}" | "[" | "]"; index: number }> = [];
  let inString = false;
  let escaped = false;
  for (let i = 0; i < text.length; i += 1) {
    const ch = text[i];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (ch === "\\") {
      escaped = inString;
      continue;
    }
    if (ch === "\"") {
      inString = !inString;
      continue;
    }
    if (!inString && (ch === "{" || ch === "}" || ch === "[" || ch === "]")) {
      out.push({ char: ch, index: i });
    }
  }
  return out;
}
