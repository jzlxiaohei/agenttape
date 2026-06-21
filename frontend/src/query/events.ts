import { useMutation, useQuery } from "@tanstack/react-query";
import { fetchSessionEvents, fetchEventDetail, fetchRaw, replayEvent } from "@/api/events";

export function useSessionEvents(sessionId: string | null) {
  return useQuery({
    queryKey: ["events", sessionId],
    queryFn: () => fetchSessionEvents(sessionId!),
    enabled: !!sessionId,
    refetchInterval: 5000,
  });
}

export function useEventDetail(eventId: string | null) {
  return useQuery({
    queryKey: ["event", eventId],
    queryFn: () => fetchEventDetail(eventId!),
    enabled: !!eventId,
  });
}

export function useRawFile(eventId: string | null, role: string, enabled: boolean) {
  return useQuery({
    queryKey: ["raw", eventId, role],
    queryFn: () => fetchRaw(eventId!, role),
    enabled: enabled && !!eventId,
  });
}

// useReplay sends a (possibly edited) request to upstream. A real billed call,
// so it is a mutation triggered explicitly — never on render.
export function useReplay(eventId: string) {
  return useMutation({
    mutationFn: (body?: string) => replayEvent(eventId, body),
  });
}
