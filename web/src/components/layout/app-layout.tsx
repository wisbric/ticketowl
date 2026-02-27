import { type ReactNode, useState, useEffect } from "react";
import { useRouterState, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/contexts/auth-context";
import { Sidebar } from "./sidebar";

interface AppLayoutProps {
  children: ReactNode;
}

const DETAIL_PATTERNS = [
  /^\/tickets\/[^/]+$/,
];

export function AppLayout({ children }: AppLayoutProps) {
  const { isLoading, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const isDetailPage = DETAIL_PATTERNS.some((p) => p.test(pathname));
  const [manualCollapse, setManualCollapse] = useState<boolean | null>(null);
  const [prevIsDetail, setPrevIsDetail] = useState(isDetailPage);

  useEffect(() => {
    if (!isLoading && !isAuthenticated && !import.meta.env.DEV) {
      navigate({ to: "/login" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  // Show loading screen while auth is being validated.
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

  // Reset manual override when route changes between detail/non-detail.
  if (prevIsDetail !== isDetailPage) {
    setPrevIsDetail(isDetailPage);
    setManualCollapse(null);
  }

  const collapsed = manualCollapse ?? isDetailPage;

  return (
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      <Sidebar collapsed={collapsed} onToggle={() => setManualCollapse(!collapsed)} />
      <main className="flex-1 overflow-y-auto p-6">{children}</main>
    </div>
  );
}
