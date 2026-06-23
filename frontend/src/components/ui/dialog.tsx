import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

export const Dialog = DialogPrimitive.Root;
export const DialogTrigger = DialogPrimitive.Trigger;
export const DialogClose = DialogPrimitive.Close;

export function DialogContent({
  className,
  children,
  closeLabel,
  onInteractOutside,
}: {
  className?: string;
  children: React.ReactNode;
  closeLabel: string;
  onInteractOutside?: React.ComponentProps<typeof DialogPrimitive.Content>["onInteractOutside"];
}) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-40 bg-black/30" />
      <DialogPrimitive.Content
        onInteractOutside={onInteractOutside}
        className={cn(
          "fixed left-1/2 top-1/2 z-50 flex max-h-[92vh] w-[min(1040px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-lg border bg-surface shadow-xl",
          className,
        )}
      >
        <DialogPrimitive.Close
          className="absolute right-3 top-3 z-10 rounded-md p-1 text-muted-foreground hover:bg-muted"
          aria-label={closeLabel}
        >
          <X size={16} />
        </DialogPrimitive.Close>
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export function DialogA11y({ title }: { title: string }) {
  return <DialogPrimitive.Title className="sr-only">{title}</DialogPrimitive.Title>;
}
