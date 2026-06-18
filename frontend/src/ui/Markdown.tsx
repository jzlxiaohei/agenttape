import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

// Default export so it can be lazy-loaded (only when the user turns on rendering;
// keeps react-markdown out of the main bundle). Raw HTML/XML is NOT parsed —
// react-markdown emits it as literal text, so prompt-engineering tags like
// <system-reminder> stay visible (next.md fidelity concern).
export default function Markdown({ text }: { text: string }) {
  return (
    <div className="md text-sm leading-relaxed">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
    </div>
  );
}
