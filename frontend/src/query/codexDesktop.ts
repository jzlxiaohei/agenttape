import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchCodexDesktopStatus, installCodexDesktop, restoreCodexDesktop } from "@/api/codexDesktop";

export function useCodexDesktopStatus() {
  return useQuery({ queryKey: ["codex-desktop-status"], queryFn: fetchCodexDesktopStatus });
}

// Install/restore mutate global config — real, billed-adjacent side effects, hence
// mutations; both refresh status so the panel reflects the new state.
export function useInstallCodexDesktop() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (hooks: boolean) => installCodexDesktop(hooks),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["codex-desktop-status"] }),
  });
}

export function useRestoreCodexDesktop() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => restoreCodexDesktop(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["codex-desktop-status"] }),
  });
}
