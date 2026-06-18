import { api } from "./client";

export interface SearchResult {
  event_id: string;
  session_id: string;
  client: string;
  provider: string;
  model: string;
  started_at: string;
  snippet: string;
}

export interface Facets {
  providers: string[];
  clients: string[];
  tags: string[];
}

export function fetchSearch(params: {
  q: string;
  tag: string;
  provider: string;
  client: string;
}): Promise<SearchResult[]> {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.tag) qs.set("tag", params.tag);
  if (params.provider) qs.set("provider", params.provider);
  if (params.client) qs.set("client", params.client);
  return api.getJSON<SearchResult[]>(`/api/search?${qs.toString()}`).then((r) => r ?? []);
}

export function fetchFacets(): Promise<Facets> {
  return api.getJSON<Facets>("/api/facets");
}
