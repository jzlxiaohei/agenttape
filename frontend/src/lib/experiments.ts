// Hands-on experiments shown in the replay library as special (non-runnable)
// instruction cards. Unlike a seed (one re-sendable request), these capture
// behavior that only shows up live with hooks on — sub-agent orchestration,
// compaction before/after — so the card just guides the user to run it
// themselves. Static config; all copy lives in i18n under experiments.<id>.*.
export interface ExperimentStep {
  note: boolean; // has an italic explanatory note line
  prompt: boolean; // has a copy-pasteable prompt block
}

export interface Experiment {
  id: "subagent" | "compaction";
  client: "cc"; // both are Claude Code-only behaviors → provider "anthropic"
  steps: ExperimentStep[];
}

export const experiments: Experiment[] = [
  {
    id: "subagent",
    client: "cc",
    steps: [
      { note: true, prompt: false },
      { note: true, prompt: true },
      { note: true, prompt: false },
      { note: true, prompt: false },
    ],
  },
  {
    id: "compaction",
    client: "cc",
    steps: [
      { note: true, prompt: false },
      { note: true, prompt: false },
      { note: true, prompt: false },
      { note: true, prompt: false },
    ],
  },
];

// visibleExperiments filters by the active provider chip. Experiments are cc-only
// (provider "anthropic"); hide them when filtering to another provider.
export function visibleExperiments(provider: string): Experiment[] {
  if (provider && provider !== "anthropic") return [];
  return experiments;
}
