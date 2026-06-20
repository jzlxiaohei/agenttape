import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

// Right-side slide-over sheet (shadcn pattern, backed by Radix Dialog so focus
// trap / Esc / scroll-lock / aria are handled for us). Used for flow-node detail.
export const Sheet = DialogPrimitive.Root;
export const SheetTrigger = DialogPrimitive.Trigger;
export const SheetClose = DialogPrimitive.Close;

export function SheetContent({
  className,
  children,
  width = "w-[clamp(640px,46vw,920px)]",
}: {
  className?: string;
  children: React.ReactNode;
  width?: string;
}) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="sheet-overlay fixed inset-0 z-40 bg-black/30" />
      <DialogPrimitive.Content
        className={cn(
          "sheet-content fixed inset-y-0 right-0 z-50 flex max-w-[92vw] flex-col border-l bg-surface shadow-xl",
          width,
          className,
        )}
      >
        <DialogPrimitive.Close
          className="absolute right-3 top-3 z-10 rounded-md p-1 text-muted-foreground hover:bg-muted"
          aria-label="Close"
        >
          <X size={16} />
        </DialogPrimitive.Close>
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

// Hidden title/description keep Radix's a11y contract satisfied without visible chrome.
export function SheetA11y({ title }: { title: string }) {
  return (
    <DialogPrimitive.Title className="sr-only">{title}</DialogPrimitive.Title>
  );
}
