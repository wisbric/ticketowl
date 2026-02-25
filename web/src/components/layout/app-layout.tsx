import { type ReactNode, useState } from "react";
import { useRouterState } from "@tanstack/react-router";
import { Sidebar } from "./sidebar";

interface AppLayoutProps {
  children: ReactNode;
}

const DETAIL_PATTERNS = [
  /^\/tickets\/[^/]+$/,
];

export function AppLayout({ children }: AppLayoutProps) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const isDetailPage = DETAIL_PATTERNS.some((p) => p.test(pathname));
  const [manualCollapse, setManualCollapse] = useState<boolean | null>(null);
  const [prevIsDetail, setPrevIsDetail] = useState(isDetailPage);

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
