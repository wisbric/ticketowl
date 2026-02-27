import { type ReactNode, useEffect } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import { Moon, Sun, LogOut } from "lucide-react";
import { useTheme } from "@/hooks/use-theme";
import { useAuth } from "@/contexts/auth-context";

interface PortalLayoutProps {
  children: ReactNode;
}

export function PortalLayout({ children }: PortalLayoutProps) {
  const { theme, toggle } = useTheme();
  const { user, logout, isLoading, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const showLogout = !import.meta.env.DEV;

  useEffect(() => {
    if (!isLoading && !isAuthenticated && !import.meta.env.DEV) {
      navigate({ to: "/login" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading && !import.meta.env.DEV) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    );
  }

  if (!isAuthenticated && !import.meta.env.DEV) {
    return null;
  }

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="flex items-center justify-between border-b border-border px-6 py-3">
        <Link to="/portal/tickets" className="flex items-center gap-2 font-semibold">
          <img src="/owl-logo.png" alt="TicketOwl" className="h-7 brightness-0 dark:brightness-100" />
          <span>TicketOwl</span>
        </Link>

        <div className="flex items-center gap-2">
          <button
            onClick={toggle}
            className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            title={theme === "dark" ? "Light mode" : "Dark mode"}
          >
            {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>
          {user && (
            <span className="text-sm text-muted-foreground">{user.display_name}</span>
          )}
          {showLogout && (
            <button
              onClick={logout}
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-red-400"
              title="Sign out"
            >
              <LogOut className="h-4 w-4" />
            </button>
          )}
        </div>
      </header>

      <main className="flex-1 p-6">{children}</main>

      <footer className="border-t border-border px-6 py-3 text-center text-xs text-muted-foreground">
        TicketOwl — A Wisbric product
      </footer>
    </div>
  );
}
