import { useQuery } from "@tanstack/react-query";
import { fetchSessions } from "@/api/sessions";

// The only place that invokes the sessions API. Caching/retry/loading is
// owned here, never in components.
export function useSessions() {
  return useQuery({
    queryKey: ["sessions"],
    queryFn: fetchSessions,
    refetchInterval: 5000, // sessions arrive live while capturing
  });
}
