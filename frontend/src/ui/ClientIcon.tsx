import { Sparkles, SquareTerminal, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

// Single source of truth for how each coding-agent client looks across the app:
// brand glyph + token color. Accepts either the stored form ("claude_code" /
// "codex_cli") or the launch kind ("cc" / "codex").
type Client = "claude_code" | "codex_cli";

function normalize(c: string): Client {
  return c === "cc" || c === "claude_code" ? "claude_code" : "codex_cli";
}

const glyph: Record<Client, LucideIcon> = {
  claude_code: Sparkles,
  codex_cli: SquareTerminal,
};

const textColor: Record<Client, string> = {
  claude_code: "text-claude",
  codex_cli: "text-codex",
};

const bgColor: Record<Client, string> = {
  claude_code: "bg-claude",
  codex_cli: "bg-codex",
};

// Inline glyph in the client's brand color.
export function ClientIcon({ client, size = 14, className }: { client: string; size?: number; className?: string }) {
  const c = normalize(client);
  const Glyph = glyph[c];
  return <Glyph size={size} className={cn(textColor[c], className)} />;
}

// Filled round avatar (white glyph on brand background) for conversation-list rows.
export function ClientAvatar({ client, size = 18, className }: { client: string; size?: number; className?: string }) {
  const c = normalize(client);
  const Glyph = glyph[c];
  return (
    <div className={cn("flex items-center justify-center rounded-full text-white", bgColor[c], className)}>
      <Glyph size={size} />
    </div>
  );
}
