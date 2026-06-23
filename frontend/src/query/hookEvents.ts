import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchHookEvents,
  addHookEvent,
  setHookEventEnabled,
  deleteHookEvent,
} from "@/api/hookEvents";

export function useHookEvents() {
  return useQuery({ queryKey: ["hook-events"], queryFn: fetchHookEvents });
}

export function useAddHookEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { client: string; event: string }) => addHookEvent(v.client, v.event),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["hook-events"] }),
  });
}

export function useSetHookEventEnabled() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { client: string; event: string; enabled: boolean }) =>
      setHookEventEnabled(v.client, v.event, v.enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["hook-events"] }),
  });
}

export function useDeleteHookEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { client: string; event: string }) => deleteHookEvent(v.client, v.event),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["hook-events"] }),
  });
}
