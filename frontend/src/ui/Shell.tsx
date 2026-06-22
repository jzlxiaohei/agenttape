import { useLocation, useNavigate } from "react-router-dom";
import { MessagesSquare, Search, Rocket, FlaskConical, BarChart3, Settings } from "lucide-react";
import { LanguageToggle } from "./LanguageToggle";
import { cn } from "@/lib/utils";

// App shell: a persistent icon rail (route navigation) wrapping page content.
export function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen">
      <IconRail />
      <div className="flex min-w-0 flex-1">{children}</div>
    </div>
  );
}

function IconRail() {
  const nav = useNavigate();
  const { pathname } = useLocation();
  const items = [
    { icon: MessagesSquare, to: "/sessions", match: (p: string) => p === "/" || p.startsWith("/sessions") },
    { icon: Rocket, to: "/launch", match: (p: string) => p.startsWith("/launch") },
    { icon: FlaskConical, to: "/cases", match: (p: string) => p.startsWith("/cases") },
    { icon: Search, to: "/search", match: (p: string) => p.startsWith("/search") },
  ];
  return (
    <nav className="flex w-14 shrink-0 flex-col items-center gap-1 border-r bg-rail py-3">
      <div className="mb-2 flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-sm font-bold text-accent-foreground">
        t
      </div>
      {items.map(({ icon: Icon, to, match }) => (
        <button
          key={to}
          onClick={() => nav(to)}
          className={cn(
            "flex h-10 w-10 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted",
            match(pathname) && "bg-accent/12 text-accent",
          )}
        >
          <Icon size={18} />
        </button>
      ))}
      <BarChart3 size={18} className="mt-1 text-muted-foreground/40" />
      <Settings size={18} className="mt-1 text-muted-foreground/40" />
      <div className="mt-auto">
        <LanguageToggle />
      </div>
    </nav>
  );
}
