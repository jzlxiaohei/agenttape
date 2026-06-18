import { useTranslation } from "react-i18next";
import { Shell } from "@/ui/Shell";
import { SessionList } from "@/ui/SessionList";
import { SessionDetail } from "@/ui/SessionDetail";

// Sessions page: list column + detail main, inside the shared shell.
export function SessionsPage() {
  const { t } = useTranslation();
  return (
    <Shell>
      <aside className="flex w-80 shrink-0 flex-col border-r bg-surface">
        <header className="px-4 py-3">
          <h1 className="text-base font-semibold">{t("sessions.title")}</h1>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto">
          <SessionList />
        </div>
      </aside>
      <main className="min-w-0 flex-1 overflow-y-auto bg-background">
        <SessionDetail />
      </main>
    </Shell>
  );
}
