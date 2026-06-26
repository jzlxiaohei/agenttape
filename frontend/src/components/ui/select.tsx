import { useState, type ReactNode } from "react";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export interface SelectOption {
  value: string;
  label: ReactNode;
  disabled?: boolean;
}

export function Select({
  value,
  options,
  placeholder,
  onChange,
  disabled,
  className,
  buttonClassName,
  menuClassName,
  optionClassName,
}: {
  value: string;
  options: SelectOption[];
  placeholder?: ReactNode;
  onChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
  buttonClassName?: string;
  menuClassName?: string;
  optionClassName?: string;
}) {
  const [open, setOpen] = useState(false);
  const selected = options.find((option) => option.value === value);
  const display = selected?.label ?? placeholder;

  const choose = (next: string) => {
    onChange(next);
    setOpen(false);
  };

  return (
    <div className={cn("relative", className)}>
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
          if (event.key === "ArrowDown" || event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            setOpen(true);
          }
        }}
        className={cn(
          "flex min-h-8 w-full items-center justify-between gap-2 rounded-md border bg-card px-2.5 py-1.5 text-left text-sm shadow-sm transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50",
          !selected && "text-muted-foreground",
          buttonClassName,
        )}
      >
        <span className="min-w-0 flex-1 truncate">{display}</span>
        <ChevronDown size={15} className={cn("shrink-0 text-muted-foreground transition-transform", open && "rotate-180")} />
      </button>
      {open && !disabled && (
        <>
          <div className="fixed inset-0 z-30" onClick={() => setOpen(false)} />
          <div
            role="listbox"
            className={cn(
              "absolute left-0 right-0 top-full z-40 mt-1 max-h-72 overflow-auto rounded-lg border bg-card p-1 shadow-lg",
              menuClassName,
            )}
          >
            {options.map((option) => (
              <button
                key={option.value}
                type="button"
                role="option"
                aria-selected={option.value === value}
                disabled={option.disabled}
                onClick={() => choose(option.value)}
                className={cn(
                  "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50",
                  option.value === value && "bg-accent/8 text-accent",
                  optionClassName,
                )}
              >
                <Check size={14} className={cn("shrink-0", option.value !== value && "opacity-0")} />
                <span className="min-w-0 flex-1 truncate">{option.label}</span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
