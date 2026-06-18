import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { fetchSearch, fetchFacets } from "@/api/search";

export function useSearch(params: { q: string; tag: string; provider: string; client: string }) {
  return useQuery({
    queryKey: ["search", params],
    queryFn: () => fetchSearch(params),
    placeholderData: keepPreviousData, // avoid flicker while typing
  });
}

export function useFacets() {
  return useQuery({ queryKey: ["facets"], queryFn: fetchFacets, staleTime: 30000 });
}
