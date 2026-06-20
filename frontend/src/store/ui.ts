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

// Cross-component UI/business state (frontend-mvvm §1). Components never hold
// this in useState.
interface UIState {
  selectedSessionId: string | null;
  selectedEventId: string | null;
  sheetEventId: string | null; // event shown in the flow detail side sheet (null = closed)
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
  timelineMode: "requests" | "timeline";

  selectSession: (id: string | null) => void;
  selectEvent: (id: string | null) => void;
  openEvent: (sessionId: string, eventId: string) => void;
  openSheet: (eventId: string) => void;
  closeSheet: () => void;
  setSearchQuery: (q: string) => void;
  setSearchFilter: (key: "searchTag" | "searchProvider" | "searchClient", value: string) => void;
  setTimelineMode: (m: "requests" | "timeline") => void;
  setDetailTab: (t: DetailTab) => void;
  togglePart: (p: DetailPart) => void;
  toggleBlock: (b: BlockKind) => void;
  setRenderMarkdown: (v: boolean) => void;
  toggleGroupRounds: () => void;
  toggleCollapsed: (key: string) => void;
}

export const useUIStore = create<UIState>((set) => ({
  selectedSessionId: null,
  selectedEventId: null,
  sheetEventId: null,
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
  timelineMode: "requests",

  selectSession: (id) => set({ selectedSessionId: id, selectedEventId: null, sheetEventId: null }),
  selectEvent: (id) => set({ selectedEventId: id }),
  openEvent: (sessionId, eventId) => set({ selectedSessionId: sessionId, selectedEventId: eventId }),
  openSheet: (eventId) => set({ sheetEventId: eventId }),
  closeSheet: () => set({ sheetEventId: null }),
  setSearchQuery: (q) => set({ searchQuery: q }),
  setSearchFilter: (key, value) => set({ [key]: value } as Partial<UIState>),
  setTimelineMode: (m) => set({ timelineMode: m }),
  setDetailTab: (t) => set({ detailTab: t }),
  togglePart: (p) => set((s) => ({ parts: { ...s.parts, [p]: !s.parts[p] } })),
  toggleBlock: (b) => set((s) => ({ blocks: { ...s.blocks, [b]: !s.blocks[b] } })),
  setRenderMarkdown: (v) => set({ renderMarkdown: v }),
  toggleGroupRounds: () => set((s) => ({ groupRounds: !s.groupRounds })),
  toggleCollapsed: (key) =>
    set((s) => ({ collapsed: { ...s.collapsed, [key]: !s.collapsed[key] } })),
}));
