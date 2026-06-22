import { useMutation, useQuery } from "@tanstack/react-query";
import { launchAgent, fetchTerminals, previewLaunch, type LaunchReq } from "@/api/launch";

// useLaunch starts an agent — an explicit action (spawns a local process), so it
// is a mutation, never run on render.
export function useLaunch() {
  return useMutation({ mutationFn: (req: LaunchReq) => launchAgent(req) });
}

export function useTerminals() {
  return useQuery({ queryKey: ["terminals"], queryFn: fetchTerminals });
}

// useLaunchPreview returns the copy-paste command + whether server launch is on.
export function useLaunchPreview(req: LaunchReq) {
  return useQuery({
    queryKey: ["launch-preview", req.kind, req.mode, req.workdir],
    queryFn: () => previewLaunch(req),
  });
}
