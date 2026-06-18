import { useTranslation } from "react-i18next";
import { ArrowLeftRight } from "lucide-react";
import type { Message } from "@/api/events";
import { ContentBlocks } from "./ContentBlocks";
import { cn } from "@/lib/utils";

// The conversation rendered as a chat thread. With markSteps, a weak divider is
// inserted whenever the flow returns from tool/client back to the assistant —
// i.e. the start of another client↔server round-trip within the same round.
export function MessageThread({ messages, markSteps }: { messages: Message[]; markSteps?: boolean }) {
  const { t } = useTranslation();
  if (messages.length === 0) return null;

  let step = 0;
  return (
    <div className="space-y-5">
      {messages.map((m, i) => {
        const boundary = !!markSteps && m.role === "assistant" && i > 0 && messages[i - 1].role !== "assistant";
        if (boundary) step += 1;
        return (
          <div key={i}>
            {boundary && step >= 2 && <StepDivider n={step} label={t("round.exchange", { n: step })} />}
            <MessageBubble message={m} />
          </div>
        );
      })}
    </div>
  );
}

function StepDivider({ n, label }: { n: number; label: string }) {
  return (
    <div className="my-4 flex items-center gap-3" aria-label={`exchange ${n}`}>
      <div className="h-px flex-1 bg-accent/25" />
      <span className="inline-flex items-center gap-1.5 rounded-full border border-accent/30 bg-accent/10 px-2.5 py-0.5 text-xs font-medium text-accent">
        <ArrowLeftRight size={12} />
        {label}
      </span>
      <div className="h-px flex-1 bg-accent/25" />
    </div>
  );
}

const roleStyle: Record<string, { avatar: string; label: string }> = {
  system: { avatar: "bg-muted-foreground", label: "text-muted-foreground" },
  user: { avatar: "bg-accent", label: "text-accent" },
  assistant: { avatar: "bg-reasoning", label: "text-reasoning" },
  tool: { avatar: "bg-toolresult", label: "text-toolresult" },
};

function MessageBubble({ message }: { message: Message }) {
  const { t } = useTranslation();
  const style = roleStyle[message.role] ?? roleStyle.system;
  const roleLabel = t(`role.${message.role}`, { defaultValue: message.role });
  return (
    <div className="flex gap-3">
      <div
        className={cn(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold uppercase text-white",
          style.avatar,
        )}
      >
        {message.role.charAt(0)}
      </div>
      <div className="min-w-0 flex-1">
        <div className={cn("mb-1 text-xs font-semibold uppercase tracking-wide", style.label)}>
          {roleLabel}
        </div>
        <div className="rounded-xl border bg-card px-3 py-2">
          <ContentBlocks blocks={message.content} />
        </div>
      </div>
    </div>
  );
}
