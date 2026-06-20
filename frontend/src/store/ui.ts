import { create } from "zustand";

export type DetailPart = "system" | "tools" | "messages";
export type DetailTab = "request" | "response" | "raw" | "diff";
export type BlockKind = "text" | "reasoning" | "tool_call" | "tool_result";

const allParts: Record<DetailPart, boolean> = {
  system: true,
  tools: true,
  messages: true,
};
const allBlocks: Record<BlockKind, boolean> = {
  text: true,
  reasoning: true,
  tool_call: true,
  tool_result: true,
};

// Cross-component UI state that is NOT navigation (frontend-mvvm §1). Navigation
// state — selected session, tab, focused request/turn — lives in the URL
// (viewmodel/route.ts), so it is shareable and survives reload. Components never
// hold any of this in useState.
interface UIState {
  detailTab: DetailTab;
  // detail filters
  parts: Record<DetailPart, boolean>;
  blocks: Record<BlockKind, boolean>;
  renderMarkdown: boolean;
  groupRounds: boolean;
  collapsed: Record<string, boolean>; // section key -> user override
  // search
  searchQuery: string;
  searchTag: string;
  searchProvider: string;
  searchClient: string;

  setSearchQuery: (q: string) => void;
  setSearchFilter: (key: "searchTag" | "searchProvider" | "searchClient", value: string) => void;
  setDetailTab: (t: DetailTab) => void;
  togglePart: (p: DetailPart) => void;
  toggleBlock: (b: BlockKind) => void;
  setRenderMarkdown: (v: boolean) => void;
  toggleGroupRounds: () => void;
  toggleCollapsed: (key: string) => void;
}

export const useUIStore = create<UIState>((set) => ({
  detailTab: "request",
  parts: { ...allParts },
  blocks: { ...allBlocks },
  renderMarkdown: false,
  groupRounds: true,
  collapsed: {},
  searchQuery: "",
  searchTag: "",
  searchProvider: "",
  searchClient: "",

  setSearchQuery: (q) => set({ searchQuery: q }),
  setSearchFilter: (key, value) => set({ [key]: value } as Partial<UIState>),
  setDetailTab: (t) => set({ detailTab: t }),
  togglePart: (p) => set((s) => ({ parts: { ...s.parts, [p]: !s.parts[p] } })),
  toggleBlock: (b) => set((s) => ({ blocks: { ...s.blocks, [b]: !s.blocks[b] } })),
  setRenderMarkdown: (v) => set({ renderMarkdown: v }),
  toggleGroupRounds: () => set((s) => ({ groupRounds: !s.groupRounds })),
  toggleCollapsed: (key) =>
    set((s) => ({ collapsed: { ...s.collapsed, [key]: !s.collapsed[key] } })),
}));
