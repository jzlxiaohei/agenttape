import { useLayoutEffect, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";

// Lightweight confirm-on-action popover: the trigger opens a small card asking to
// confirm a destructive/billed action, instead of a clumsy in-place two-click arm.
// Pure view-local open state. Trigger is a render-prop so callers keep their own
// button styling (primary Run button, list-row trash icon, …).
export function Popconfirm({
  message,
  confirmLabel,
  cancelLabel,
  tone = "default",
  placement = "top",
  disabled,
  onConfirm,
  children,
}: {
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  tone?: "default" | "danger";
  placement?: "top" | "bottom" | "right";
  disabled?: boolean;
  onConfirm: () => void;
  children: (api: { open: () => void; isOpen: boolean }) => React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [style, setStyle] = useState<CSSProperties>({});
  const triggerRef = useRef<HTMLSpanElement>(null);
  const open = () => {
    if (!disabled) setIsOpen(true);
  };

  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const width = 256;
    const gap = 8;
    const left = (x: number) => Math.min(Math.max(12, x), window.innerWidth - width - 12);
    if (placement === "right") {
      setStyle({ left: left(rect.right + gap), top: Math.min(rect.top, window.innerHeight - 160) });
      return;
    }
    if (placement === "bottom") {
      setStyle({ left: left(rect.left + rect.width / 2 - width / 2), top: rect.bottom + gap });
      return;
    }
    setStyle({ left: left(rect.left + rect.width / 2 - width / 2), top: Math.max(12, rect.top - 156) });
  }, [isOpen, placement]);

  return (
    <span ref={triggerRef} className="relative inline-flex">
      {children({ open, isOpen })}
      {isOpen && (
        createPortal(
          <>
          {/* data-popconfirm + pointer-events-auto let this work even inside a
              Radix modal Dialog, which otherwise locks pointer events on body and
              would dismiss the dialog on this outside click (see DialogContent). */}
          <div
            data-popconfirm
            className="pointer-events-auto fixed inset-0 z-[55]"
            onClick={(e) => { e.stopPropagation(); setIsOpen(false); }}
          />
          <div
            data-popconfirm
            style={style}
            onClick={(e) => e.stopPropagation()}
            className="pointer-events-auto fixed z-[60] w-64 rounded-lg border bg-card p-3 shadow-lg"
          >
            <p className="text-sm leading-relaxed text-foreground">{message}</p>
            <div className="mt-3 flex justify-end gap-2">
              <button
                onClick={() => setIsOpen(false)}
                className="rounded-md px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted"
              >
                {cancelLabel}
              </button>
              <button
                onClick={() => {
                  setIsOpen(false);
                  onConfirm();
                }}
                className={cn(
                  "rounded-md px-3 py-1.5 text-sm font-medium text-white",
                  tone === "danger" ? "bg-error" : "bg-accent",
                )}
              >
                {confirmLabel}
              </button>
            </div>
          </div>
          </>,
          document.body,
        )
      )}
    </span>
  );
}
