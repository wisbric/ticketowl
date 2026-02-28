import { Link, useMatchRoute } from "@tanstack/react-router";
import { useState, useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Inbox,
  Clock,
  Settings,
  Info,
  Moon,
  Sun,
  LogOut,
  ExternalLink,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "@/hooks/use-theme";
import { useAuth } from "@/contexts/auth-context";

const navItems = [
  { to: "/", label: "The Perch", icon: Inbox },
  { to: "/sla", label: "The Watch", icon: Clock },
] as const;

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
}

function getAvatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  const colors = [
    "bg-emerald-600",
    "bg-teal-600",
    "bg-cyan-600",
    "bg-sky-600",
    "bg-blue-600",
    "bg-indigo-600",
    "bg-violet-600",
    "bg-purple-600",
  ];
  return colors[Math.abs(hash) % colors.length];
}

interface SidebarProps {
  collapsed?: boolean;
  onToggle?: () => void;
}

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const matchRoute = useMatchRoute();
  const { theme, toggle } = useTheme();
  const { user, logout } = useAuth();
  const showLogout = !import.meta.env.DEV;
  const [showUserMenu, setShowUserMenu] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);

  const { data: authConfig } = useQuery({
    queryKey: ["auth-config"],
    queryFn: async () => {
      const res = await fetch("/auth/config");
      if (!res.ok) return {};
      return res.json();
    },
    staleTime: Infinity,
  });

  useEffect(() => {
    if (!showUserMenu) return;
    const handler = (e: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setShowUserMenu(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showUserMenu]);

  if (collapsed) {
    return (
      <aside className="flex w-14 shrink-0 flex-col items-center border-r border-border bg-sidebar py-3">
        <Link to="/" className="mb-2" title="Home">
          <img src="/owl-logo.png" alt="TicketOwl" className="h-7 brightness-0 dark:brightness-100" />
        </Link>

        <nav className="flex flex-1 flex-col items-center gap-1">
          {navItems.map((item) => {
            const active = item.to === "/"
              ? matchRoute({ to: "/", fuzzy: false })
              : matchRoute({ to: item.to, fuzzy: true });
            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  "rounded-md p-2 transition-colors",
                  active
                    ? "bg-muted text-accent"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
                title={item.label}
              >
                <item.icon className="h-4 w-4" />
              </Link>
            );
          })}
        </nav>

        <div className="mt-auto flex flex-col items-center gap-1 pt-2">
          {onToggle && (
            <button
              onClick={onToggle}
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              title="Expand sidebar"
            >
              <PanelLeftOpen className="h-4 w-4" />
            </button>
          )}
          <button
            onClick={toggle}
            className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            title={theme === "dark" ? "Light mode" : "Dark mode"}
          >
            {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>
          {user && (
            <div className="relative" ref={userMenuRef}>
              <button
                onClick={() => setShowUserMenu(!showUserMenu)}
                className="rounded-md p-1 transition-colors hover:bg-muted"
                title={user.display_name}
              >
                <div
                  className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-medium text-white ${getAvatarColor(user.display_name)}`}
                >
                  {getInitials(user.display_name)}
                </div>
              </button>
              {showUserMenu && (
                <div className="absolute bottom-full left-full z-50 mb-1 ml-1 w-48 rounded-lg border border-border bg-card shadow-lg">
                  <div className="border-b border-border px-3 py-2.5">
                    <div className="truncate text-sm font-medium text-foreground">
                      {user.display_name}
                    </div>
                    <div className="truncate text-xs text-muted-foreground">{user.email}</div>
                  </div>
                  <div className="py-1">
                    <button
                      onClick={toggle}
                      className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                    >
                      {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
                      {theme === "dark" ? "Light mode" : "Dark mode"}
                    </button>
                  </div>
                  {(authConfig?.nightowl_url || authConfig?.bookowl_url) && (
                    <div className="border-t border-border py-1">
                      {authConfig.nightowl_url && (
                        <a
                          href={authConfig.nightowl_url}
                          className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                        >
                          <ExternalLink className="h-4 w-4" /> NightOwl
                        </a>
                      )}
                      {authConfig.bookowl_url && (
                        <a
                          href={authConfig.bookowl_url}
                          className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                        >
                          <ExternalLink className="h-4 w-4" /> BookOwl
                        </a>
                      )}
                    </div>
                  )}
                  {showLogout && (
                    <div className="border-t border-border py-1">
                      <button
                        onClick={() => { setShowUserMenu(false); logout(); }}
                        className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-red-400 transition-colors hover:bg-muted"
                      >
                        <LogOut className="h-4 w-4" /> Sign out
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
          {user?.role === "admin" && (
            <Link
              to="/admin"
              className={cn(
                "rounded-md p-2 transition-colors",
                matchRoute({ to: "/admin", fuzzy: true })
                  ? "bg-muted text-accent"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
              title="Admin"
            >
              <Settings className="h-4 w-4" />
            </Link>
          )}
          <Link
            to="/about"
            className={cn(
              "rounded-md p-2 transition-colors",
              matchRoute({ to: "/about", fuzzy: false })
                ? "bg-muted text-accent"
                : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
            title="About"
          >
            <Info className="h-4 w-4" />
          </Link>
        </div>
      </aside>
    );
  }

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-border bg-sidebar">
      <div className="flex items-center border-b border-border px-4 py-3">
        <Link to="/" className="flex items-center gap-2 font-semibold text-sidebar-foreground">
          <img src="/owl-logo.png" alt="TicketOwl" className="h-8 brightness-0 dark:brightness-100" />
          <span>TicketOwl</span>
        </Link>
      </div>

      <nav className="flex-1 space-y-0.5 px-3 py-3">
        {navItems.map((item) => {
          const active = item.to === "/"
            ? matchRoute({ to: "/", fuzzy: false })
            : matchRoute({ to: item.to, fuzzy: true });
          return (
            <Link
              key={item.to}
              to={item.to}
              className={cn(
                "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
                active
                  ? "bg-muted text-accent"
                  : "text-sidebar-foreground hover:bg-muted hover:text-foreground"
              )}
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      {/* Collapse toggle */}
      <div className="px-3 pb-1 space-y-0.5">
        {onToggle && (
          <button
            onClick={onToggle}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            title="Collapse sidebar"
          >
            <PanelLeftClose className="h-4 w-4" />
            <span>Collapse</span>
          </button>
        )}
      </div>

      <div className="border-t border-border px-3 py-3">
        {user ? (
          <div className="relative" ref={userMenuRef}>
            <button
              onClick={() => setShowUserMenu(!showUserMenu)}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-muted"
            >
              <div
                className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-medium text-white ${getAvatarColor(user.display_name)}`}
              >
                {getInitials(user.display_name)}
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-sidebar-foreground">
                  {user.display_name}
                </div>
                <div className="truncate text-xs text-muted-foreground">
                  {user.role}
                </div>
              </div>
            </button>

            {showUserMenu && (
              <div className="absolute bottom-full left-0 right-0 z-50 mb-1 rounded-lg border border-border bg-card shadow-lg">
                <div className="border-b border-border px-3 py-2.5">
                  <div className="truncate text-sm font-medium text-foreground">
                    {user.display_name}
                  </div>
                  <div className="truncate text-xs text-muted-foreground">{user.email}</div>
                </div>

                <div className="py-1">
                  <button
                    onClick={toggle}
                    className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                  >
                    {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
                    {theme === "dark" ? "Light mode" : "Dark mode"}
                  </button>
                </div>

                {(authConfig?.nightowl_url || authConfig?.bookowl_url) && (
                  <div className="border-t border-border py-1">
                    {authConfig.nightowl_url && (
                      <a
                        href={authConfig.nightowl_url}
                        className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                      >
                        <ExternalLink className="h-4 w-4" /> NightOwl
                      </a>
                    )}
                    {authConfig.bookowl_url && (
                      <a
                        href={authConfig.bookowl_url}
                        className="flex items-center gap-2 px-3 py-1.5 text-sm text-sidebar-foreground transition-colors hover:bg-muted"
                      >
                        <ExternalLink className="h-4 w-4" /> BookOwl
                      </a>
                    )}
                  </div>
                )}

                {showLogout && (
                  <div className="border-t border-border py-1">
                    <button
                      onClick={() => {
                        setShowUserMenu(false);
                        logout();
                      }}
                      className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-red-400 transition-colors hover:bg-muted"
                    >
                      <LogOut className="h-4 w-4" />
                      Sign out
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        ) : null}
      </div>

      <div className="px-3 pb-2 space-y-0.5">
        {user?.role === "admin" && (
          <Link
            to="/admin"
            className={cn(
              "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
              matchRoute({ to: "/admin", fuzzy: true })
                ? "bg-muted text-accent"
                : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
          >
            <Settings className="h-4 w-4" />
            Admin
          </Link>
        )}
        <Link
          to="/about"
          className={cn(
            "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
            matchRoute({ to: "/about", fuzzy: false })
              ? "bg-muted text-accent"
              : "text-muted-foreground hover:bg-muted hover:text-foreground"
          )}
        >
          <Info className="h-4 w-4" />
          About
        </Link>
      </div>
    </aside>
  );
}
