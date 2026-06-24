import type { TFunction } from "i18next";
import type { ActiveSession, ReplayCase } from "@/api/cases";

export interface CaseSection {
  key: "built_in" | "added";
  cases: ReplayCase[];
}

// caseDisplayName localizes built-in (seed) titles so they aren't hardcoded to one
// language: a seed's id maps to an i18n key (cases.seed.<id>), falling back to the
// stored name. User-created cases keep their own name verbatim.
export function caseDisplayName(c: ReplayCase, t: TFunction): string {
  if (c.source !== "seed") return c.name;
  const key = `cases.seed.${c.id.replace(/^seed:/, "").replace(/-/g, "_")}`;
  return t(key, { defaultValue: c.name });
}

export function caseSections(cases: ReplayCase[]): CaseSection[] {
  const builtIn = cases.filter((c) => c.source === "seed");
  const added = cases.filter((c) => c.source !== "seed");
  // User-created cases come first; built-in seeds after.
  const sections: CaseSection[] = [
    { key: "added", cases: added },
    { key: "built_in", cases: builtIn },
  ];
  return sections.filter((s) => s.cases.length > 0);
}

// caseProviderClient maps a wire provider to the coding-agent brand used for
// icon/color (ClientIcon understands "cc" / "codex"). anthropic ⇒ Claude Code,
// everything else (openai-responses…) ⇒ codex.
export function caseProviderClient(provider: string): "cc" | "codex" {
  return provider === "anthropic" ? "cc" : "codex";
}

// caseDescription localizes the one-line experiment blurb for built-in cases
// (cases.seed_desc.<id>); user cases have no canned description.
export function caseDescription(c: ReplayCase, t: TFunction): string {
  if (c.source !== "seed") return "";
  const key = `cases.seed_desc.${c.id.replace(/^seed:/, "").replace(/-/g, "_")}`;
  return t(key, { defaultValue: "" });
}

// caseHint returns an optional "try this" callout for a seed (e.g. cc-edit's
// toolset A/B), or "" if the seed has no hint. Keyed by cases.hint_<id>.
export function caseHint(c: ReplayCase, t: TFunction): string {
  if (c.source !== "seed") return "";
  const key = `cases.hint_${c.id.replace(/^seed:/, "").replace(/-/g, "_")}`;
  return t(key, { defaultValue: "" });
}

export interface CaseCardMeta {
  model: string;
  tools: number;
  stream: boolean;
}

// caseCardMeta pulls the few "core fields" a card surfaces straight out of the
// stored request body (model, tool count, streaming). Best-effort: a body that
// doesn't parse just yields empty/zero values.
export function caseCardMeta(c: ReplayCase): CaseCardMeta {
  let body: Record<string, unknown> = {};
  try {
    const parsed = JSON.parse(c.body);
    if (parsed && typeof parsed === "object") body = parsed as Record<string, unknown>;
  } catch {
    // ignore — non-JSON body, leave defaults
  }
  const tools = Array.isArray(body.tools) ? body.tools.length : 0;
  return {
    model: typeof body.model === "string" ? body.model : "",
    tools,
    stream: body.stream === true,
  };
}

// caseProviders is the distinct, sorted provider list across cases — drives the
// provider filter chips.
export function caseProviders(cases: ReplayCase[]): string[] {
  return Array.from(new Set(cases.map((c) => c.provider).filter(Boolean))).sort();
}

export function filterCasesByProvider(cases: ReplayCase[], provider: string): ReplayCase[] {
  if (!provider) return cases;
  return cases.filter((c) => c.provider === provider);
}

// providerMatchesClient is true when a live session's client can serve a case of
// the given wire provider (anthropic ⇔ Claude Code, openai-responses ⇔ codex).
export function providerMatchesClient(provider: string, client: string): boolean {
  return caseProviderClient(provider) === (client === "claude_code" || client === "cc" ? "cc" : "codex");
}

// authTargets reduces compatible sessions to DISTINCT auth choices: sessions with
// the same (upstream, credential_kind) carry interchangeable auth, so we keep one
// representative each. A picker is only worth showing when this returns >1 — e.g.
// a subscription session AND an API-key session for the same client.
export function authTargets(sessions: ActiveSession[]): ActiveSession[] {
  const seen = new Set<string>();
  const out: ActiveSession[] = [];
  for (const s of sessions) {
    const key = `${s.upstream}|${s.credential_kind}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(s);
  }
  return out;
}

export function caseRunURL(caseItem: ReplayCase, session: ActiveSession | undefined): string {
  if (!session) return "";
  return joinURL(session.upstream, caseEndpoint(caseItem));
}

export function caseEndpoint(caseItem: ReplayCase): string {
  return caseItem.endpoint || endpointFromTarget(caseItem.target);
}

function endpointFromTarget(target: string): string {
  try {
    const u = new URL(target);
    if (u.pathname.endsWith("/responses")) return "/responses";
    return u.pathname + u.search;
  } catch {
    return target.startsWith("/") ? target : `/${target}`;
  }
}

function joinURL(base: string, endpoint: string): string {
  return `${base.replace(/\/+$/, "")}/${endpoint.replace(/^\/+/, "")}`;
}
