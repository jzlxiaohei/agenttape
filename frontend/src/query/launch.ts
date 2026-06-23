import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  launchAgent,
  fetchTerminals,
  previewLaunch,
  fetchManualCommand,
  type LaunchReq,
  type LaunchKind,
  type LaunchMode,
} from "@/api/launch";

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
    queryKey: ["launch-preview", req.kind, req.mode, req.workdir, req.args],
    queryFn: () => previewLaunch(req),
  });
}

// useManualTemplate returns the env/-c "run it yourself" command with a <TOKEN>
// placeholder — no session is registered (safe to fetch live on every change).
export function useManualTemplate(req: { kind: LaunchKind; mode?: LaunchMode; args?: string }) {
  return useQuery({
    queryKey: ["manual-command", req.kind, req.mode, req.args],
    queryFn: () => fetchManualCommand({ ...req, register: false }),
  });
}

// useGenerateManual registers a session and returns the real, ready-to-run command
// (an action, so a mutation). Invalidates active-sessions so the new session shows
// up in the replay picker.
export function useGenerateManual() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: { kind: LaunchKind; mode?: LaunchMode; args?: string }) =>
      fetchManualCommand({ ...req, register: true }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["active-sessions"] }),
  });
}
