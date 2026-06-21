import CodeMirror from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { forceParsing } from "@codemirror/language";
import type { EditorView } from "@codemirror/view";

// Editable JSON editor (CodeMirror) — line numbers, folding, bracket matching/
// closing. Default export so it loads with its (lazy) host. The read-only sibling
// is CodeViewer.
export default function CodeEditor({
  value,
  onChange,
  height = "300px",
}: {
  value: string;
  onChange: (v: string) => void;
  height?: string;
}) {
  return (
    <div className="overflow-hidden rounded-lg border">
      <CodeMirror
        value={value}
        height={height}
        editable
        extensions={[json()]}
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
  );
}
