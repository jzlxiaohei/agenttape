import { Shell } from "@/ui/Shell";
import { SessionDetail } from "@/ui/SessionDetail";

// Sessions page: the captured-session list lives in the shared shell sidebar, so
// this page is just the detail of the selected session.
export function SessionsPage() {
  return (
    <Shell>
      <main className="min-w-0 flex-1 overflow-y-auto bg-background">
        <SessionDetail />
      </main>
    </Shell>
  );
}
