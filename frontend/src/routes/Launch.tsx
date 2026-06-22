import { Shell } from "@/ui/Shell";
import { LaunchPanel } from "@/ui/LaunchPanel";

// Launch page: start a proxied agent session from the browser.
export function LaunchPage() {
  return (
    <Shell>
      <main className="min-w-0 flex-1 overflow-y-auto bg-background">
        <LaunchPanel />
      </main>
    </Shell>
  );
}
