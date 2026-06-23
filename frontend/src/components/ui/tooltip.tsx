import { useLayoutEffect, useRef, useState } from "react";
import type { CSSProperties, ReactNode } from "react";
import { createPortal } from "react-dom";

type Placement = "top" | "bottom" | "left" | "right";

// Lightweight hover tooltip: a portal-rendered dark bubble positioned next to its
// trigger. Built on the same getBoundingClientRect + portal pattern as Popconfirm
// (no extra deps, no native `title` lag). Opens after a short delay so it doesn't
// flash while the pointer passes through; closes immediately. Brief hints only —
// not for interactive content.
export function Tooltip({
  content,
  placement = "top",
  delay = 120,
  children,
}: {
  content: ReactNode;
  placement?: Placement;
  delay?: number;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [style, setStyle] = useState<CSSProperties>({});
  const triggerRef = useRef<HTMLSpanElement>(null);
  const timer = useRef<number | undefined>(undefined);

  const show = () => {
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setOpen(true), delay);
  };
  const hide = () => {
    window.clearTimeout(timer.current);
    setOpen(false);
  };

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return;
    setStyle(positionFor(placement, triggerRef.current.getBoundingClientRect()));
  }, [open, placement]);

  return (
    <span ref={triggerRef} onMouseEnter={show} onMouseLeave={hide} className="inline-flex">
      {children}
      {open &&
        content &&
        createPortal(
          <div
            role="tooltip"
            style={style}
            className="pointer-events-none fixed z-[60] max-w-xs rounded-md bg-foreground px-2 py-1 text-xs leading-snug text-background shadow-lg"
          >
            {content}
          </div>,
          document.body,
        )}
    </span>
  );
}

// Anchor the bubble to a side of the trigger using transform-based centering, so
// we don't need to measure the (variable-width) bubble first.
function positionFor(placement: Placement, r: DOMRect): CSSProperties {
  const gap = 6;
  switch (placement) {
    case "bottom":
      return { left: r.left + r.width / 2, top: r.bottom + gap, transform: "translate(-50%, 0)" };
    case "left":
      return { left: r.left - gap, top: r.top + r.height / 2, transform: "translate(-100%, -50%)" };
    case "right":
      return { left: r.right + gap, top: r.top + r.height / 2, transform: "translate(0, -50%)" };
    case "top":
    default:
      return { left: r.left + r.width / 2, top: r.top - gap, transform: "translate(-50%, -100%)" };
  }
}
