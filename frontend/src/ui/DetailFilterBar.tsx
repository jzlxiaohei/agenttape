import { useTranslation } from "react-i18next";
import { Layers, Type } from "lucide-react";
import { useUIStore, type BlockKind, type DetailPart } from "@/store/ui";
import { cn } from "@/lib/utils";

const PARTS: DetailPart[] = ["system", "tools", "messages"];
const BLOCKS: BlockKind[] = ["text", "reasoning", "tool_call", "tool_result"];

// Two distinct kinds of control:
//   • Filters (parts / blocks): hide content — off state is struck through.
//   • View modes (group rounds / markdown): change rendering, not what's shown —
//     styled as pill toggles on the right so they aren't mistaken for filters.
export function DetailFilterBar() {
  const { t } = useTranslation();
  const parts = useUIStore((s) => s.parts);
  const blocks = useUIStore((s) => s.blocks);
  const groupRounds = useUIStore((s) => s.groupRounds);
  const renderMarkdown = useUIStore((s) => s.renderMarkdown);
  const togglePart = useUIStore((s) => s.togglePart);
  const toggleBlock = useUIStore((s) => s.toggleBlock);
  const toggleGroupRounds = useUIStore((s) => s.toggleGroupRounds);
  const setRenderMarkdown = useUIStore((s) => s.setRenderMarkdown);

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-y py-2 text-xs">
      <Group label={t("filter.parts")}>
        {PARTS.map((p) => (
          <FilterChip key={p} active={parts[p]} onClick={() => togglePart(p)}>
            {t(`section.${p}`, { defaultValue: p })}
          </FilterChip>
        ))}
      </Group>
      <Group label={t("filter.blocks")}>
        {BLOCKS.map((b) => (
          <FilterChip key={b} active={blocks[b]} onClick={() => toggleBlock(b)}>
            {t(`block.${b}`, { defaultValue: b })}
          </FilterChip>
        ))}
      </Group>

      <div className="ml-auto flex items-center gap-2">
        <ModeToggle icon={<Layers size={13} />} active={groupRounds} onClick={toggleGroupRounds}>
          {t("filter.rounds")}
        </ModeToggle>
        <ModeToggle icon={<Type size={13} />} active={renderMarkdown} onClick={() => setRenderMarkdown(!renderMarkdown)}>
          {t("filter.markdown")}
        </ModeToggle>
      </div>
    </div>
  );
}

function Group({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-muted-foreground">{label}</span>
      {children}
    </div>
  );
}

// Filter: off = struck through (content hidden).
function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "rounded-md border px-2 py-0.5",
        active
          ? "border-accent/50 bg-accent/10 text-accent"
          : "border-transparent text-muted-foreground line-through opacity-60 hover:opacity-100",
      )}
    >
      {children}
    </button>
  );
}

// View mode: a pill switch (icon + on/off dot), clearly not a content filter.
function ModeToggle({
  icon,
  active,
  onClick,
  children,
}: {
  icon: React.ReactNode;
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-medium transition-colors",
        active
          ? "border-accent bg-accent text-accent-foreground"
          : "border-border text-muted-foreground hover:bg-muted",
      )}
    >
      {icon}
      {children}
      <span
        className={cn(
          "ml-0.5 h-1.5 w-1.5 rounded-full",
          active ? "bg-accent-foreground" : "bg-muted-foreground/40",
        )}
      />
    </button>
  );
}
