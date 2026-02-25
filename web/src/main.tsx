import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  RouterProvider,
  createRouter,
  createRootRoute,
  createRoute,
  Outlet,
  redirect,
} from "@tanstack/react-router";
import { AuthProvider } from "@/contexts/auth-context";
import { AppLayout } from "@/components/layout/app-layout";
import { PortalLayout } from "@/components/layout/portal-layout";
import { initTheme } from "@/hooks/use-theme";
import { LoginPage } from "@/pages/login";
import { TicketListPage } from "@/pages/ticket-list";
import { TicketDetailPage } from "@/pages/ticket-detail";
import { AdminSLAPage } from "@/pages/admin-sla";
import { AdminPage } from "@/pages/admin";
import { AdminZammadPage } from "@/pages/admin-zammad";
import { AdminIntegrationsPage } from "@/pages/admin-integrations";
import { AdminCustomersPage } from "@/pages/admin-customers";
import { AdminRulesPage } from "@/pages/admin-rules";
import { PortalTicketListPage } from "@/pages/portal-ticket-list";
import { PortalTicketDetailPage } from "@/pages/portal-ticket-detail";
import { NotFoundPage } from "@/pages/not-found";
import "./index.css";

initTheme();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1 },
  },
});

function requireAuth() {
  if (import.meta.env.DEV) return;
  const token = localStorage.getItem("ticketowl_token");
  if (!token) {
    throw redirect({ to: "/login" });
  }
}

const publicRootRoute = createRootRoute({
  component: () => <Outlet />,
});

// Authenticated layout with sidebar.
const appLayoutRoute = createRoute({
  getParentRoute: () => publicRootRoute,
  id: "app",
  beforeLoad: requireAuth,
  component: () => (
    <AppLayout>
      <Outlet />
    </AppLayout>
  ),
});

// Portal layout (minimal header, no sidebar).
const portalLayoutRoute = createRoute({
  getParentRoute: () => publicRootRoute,
  id: "portal",
  beforeLoad: requireAuth,
  component: () => (
    <PortalLayout>
      <Outlet />
    </PortalLayout>
  ),
});

// Public routes.
const loginRoute = createRoute({
  getParentRoute: () => publicRootRoute,
  path: "/login",
  component: LoginPage,
});

// Authenticated app routes.
const indexRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/",
  component: TicketListPage,
});
const ticketDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/tickets/$ticketId",
  component: TicketDetailPage,
});
const slaRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/sla",
  component: AdminSLAPage,
});
const adminRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/admin",
  component: AdminPage,
});
const adminZammadRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/admin/zammad",
  component: AdminZammadPage,
});
const adminIntegrationsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/admin/integrations",
  component: AdminIntegrationsPage,
});
const adminCustomersRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/admin/customers",
  component: AdminCustomersPage,
});
const adminRulesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/admin/rules",
  component: AdminRulesPage,
});
const notFoundRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "$",
  component: NotFoundPage,
});

// Portal routes.
const portalTicketsRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: "/portal/tickets",
  component: PortalTicketListPage,
});
const portalTicketDetailRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: "/portal/tickets/$ticketId",
  component: PortalTicketDetailPage,
});

const routeTree = publicRootRoute.addChildren([
  loginRoute,
  appLayoutRoute.addChildren([
    indexRoute,
    ticketDetailRoute,
    slaRoute,
    adminRoute,
    adminZammadRoute,
    adminIntegrationsRoute,
    adminCustomersRoute,
    adminRulesRoute,
    notFoundRoute,
  ]),
  portalLayoutRoute.addChildren([
    portalTicketsRoute,
    portalTicketDetailRoute,
  ]),
]);

const router = createRouter({ routeTree });

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>
);
