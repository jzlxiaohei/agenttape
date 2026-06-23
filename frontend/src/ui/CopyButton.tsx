import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Check, Copy } from "lucide-react";
import { cn } from "@/lib/utils";

// Copy-to-clipboard button with transient "copied" feedback. Handy for read-only
// CodeMirror views (RawView / CodeViewer) where the content can't be select-all'd.
export function CopyButton({ text, className }: { text: string; className?: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  // Reset the checkmark if the source text changes (e.g. switching request/response).
  useEffect(() => setCopied(false), [text]);

  const copy = () => {
    if (!text) return;
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };

  return (
    <button
      type="button"
      onClick={copy}
      className={cn("inline-flex items-center gap-1 text-muted-foreground hover:text-foreground", className)}
    >
      {copied ? <Check size={12} /> : <Copy size={12} />}
      {copied ? t("launch.copied") : t("launch.copy")}
    </button>
  );
}
