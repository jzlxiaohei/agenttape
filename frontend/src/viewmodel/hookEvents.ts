import type { HookEventDef } from "@/api/hookEvents";

export interface HookClientGroup {
  client: string;
  events: HookEventDef[];
  enabledCount: number;
}

// The clients we expose, in display order. Kept in sync with store.HookClients on
// the backend; a client with no rows yet still renders (empty group) so the user
// sees it exists.
const CLIENT_ORDER = ["claude_code", "codex"];

// hookClientGroups buckets the flat registry into per-client groups, events
// alphabetical, clients in a stable display order. Pure — no fetching, no React.
export function hookClientGroups(defs: HookEventDef[]): HookClientGroup[] {
  const byClient = new Map<string, HookEventDef[]>();
  for (const c of CLIENT_ORDER) byClient.set(c, []);
  for (const d of defs) {
    if (!byClient.has(d.client)) byClient.set(d.client, []);
    byClient.get(d.client)!.push(d);
  }
  return [...byClient.entries()].map(([client, events]) => {
    const sorted = [...events].sort((a, b) => a.event.localeCompare(b.event));
    return {
      client,
      events: sorted,
      enabledCount: sorted.filter((e) => e.enabled).length,
    };
  });
}
