import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchCases, fetchActiveSessions, closeActiveSession, runCase, addCase, createCase, snapshotCase, deleteCase, overwriteCase, caseCurl } from "@/api/cases";
import type { CurlMode } from "@/api/cases";

export function useCases() {
  return useQuery({ queryKey: ["cases"], queryFn: fetchCases });
}

export function useActiveSessions() {
  return useQuery({ queryKey: ["active-sessions"], queryFn: fetchActiveSessions, refetchInterval: 5000 });
}

// useCloseActiveSession forgets a live session (proxy registry + in-memory creds).
export function useCloseActiveSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => closeActiveSession(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["active-sessions"] }),
  });
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

export function useCreateCase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { name: string; tags?: string; provider: string; endpoint: string; body: string }) => createCase(v),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cases"] }),
  });
}

// useSnapshotCase saves an edited body as a new derived case.
export function useSnapshotCase(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { body: string; name?: string }) => snapshotCase(id, v.body, v.name),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cases"] }),
  });
}

export function useDeleteCase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteCase(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cases"] }),
  });
}

// useOverwriteCase saves an edited body back onto the same case (in-place).
export function useOverwriteCase(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => overwriteCase(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cases"] }),
  });
}

// useCaseCurl builds a curl on demand (a mutation, since it depends on the live
// session, the chosen mode, and whether to reveal credentials).
export function useCaseCurl(id: string) {
  return useMutation({
    mutationFn: (v: { sessionId: string; mode: CurlMode; reveal?: boolean; body?: string }) =>
      caseCurl(id, v),
  });
}
