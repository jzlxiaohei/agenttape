import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchCases, fetchActiveSessions, runCase, addCase } from "@/api/cases";

export function useCases() {
  return useQuery({ queryKey: ["cases"], queryFn: fetchCases });
}

export function useActiveSessions() {
  return useQuery({ queryKey: ["active-sessions"], queryFn: fetchActiveSessions, refetchInterval: 5000 });
}

// useRunCase runs a case against a session — a real billed call, hence a mutation.
export function useRunCase(id: string) {
  return useMutation({
    mutationFn: (v: { sessionId: string; body?: string }) => runCase(id, v.sessionId, v.body),
  });
}

export function useAddCase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { eventId: string; name?: string; tags?: string }) => addCase(v.eventId, v.name, v.tags),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cases"] }),
  });
}
