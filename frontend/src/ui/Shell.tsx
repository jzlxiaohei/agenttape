import { useLocation, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Rocket, Search, FlaskConical, Webhook } from "lucide-react";
import { LanguageToggle } from "./LanguageToggle";
import { SessionList } from "./SessionList";
import { cn } from "@/lib/utils";

// App shell: a persistent left sidebar (function nav on top, captured-session list
// below) wrapping the routed page content.
export function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex min-w-0 flex-1">{children}</div>
    </div>
  );
}

function Sidebar() {
  const nav = useNavigate();
  const { pathname } = useLocation();
  const { t } = useTranslation();
  const items = [
    { icon: Rocket, label: t("launch.title"), to: "/launch", active: pathname.startsWith("/launch") },
    { icon: Search, label: t("search.title"), to: "/search", active: pathname.startsWith("/search") },
    { icon: FlaskConical, label: t("cases.title"), to: "/cases", active: pathname.startsWith("/cases") },
    { icon: Webhook, label: t("hooks.title"), to: "/hooks", active: pathname.startsWith("/hooks") },
  ];
  return (
    <nav className="flex w-72 shrink-0 flex-col border-r bg-rail">
      <button
        onClick={() => nav("/sessions")}
        className="flex items-center gap-2 px-4 py-3 text-left"
      >
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-accent text-sm font-bold text-accent-foreground">
          t
        </div>
        <span className="font-semibold">tracelab</span>
      </button>

      <div className="space-y-0.5 px-2">
        {items.map(({ icon: Icon, label, to, active }) => (
          <button
            key={to}
            onClick={() => nav(to)}
            className={cn(
              "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-muted",
              active ? "bg-accent/12 font-medium text-accent" : "text-foreground",
            )}
          >
            <Icon size={17} className="shrink-0" />
            <span className="truncate">{label}</span>
          </button>
        ))}
      </div>

      <div className="mx-3 my-2 border-t" />
      <span className="px-4 pb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {t("sessions.title")}
      </span>
      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-2">
        <SessionList />
      </div>

      <div className="border-t px-3 py-2">
        <LanguageToggle />
      </div>
    </nav>
  );
}
