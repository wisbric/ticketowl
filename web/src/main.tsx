import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  RouterProvider,
  createRouter,
  createRootRoute,
  createRoute,
  Outlet,
} from "@tanstack/react-router";
import { AuthProvider } from "@/contexts/auth-context";
import { AppLayout } from "@/components/layout/app-layout";
import { PortalLayout } from "@/components/layout/portal-layout";
import { initTheme } from "@/hooks/use-theme";
import { LoginPage } from "@/pages/login";
import { AuthCallbackPage } from "@/pages/auth-callback";
import { TicketListPage } from "@/pages/ticket-list";
import { TicketDetailPage } from "@/pages/ticket-detail";
import { AdminSLAPage } from "@/pages/admin-sla";
import { AdminAuthPage } from "@/pages/admin-auth";
import { AdminPage } from "@/pages/admin";
import { AdminZammadPage } from "@/pages/admin-zammad";
import { AdminIntegrationsPage } from "@/pages/admin-integrations";
import { AdminCustomersPage } from "@/pages/admin-customers";
import { AdminRulesPage } from "@/pages/admin-rules";
import { PortalTicketListPage } from "@/pages/portal-ticket-list";
import { PortalTicketDetailPage } from "@/pages/portal-ticket-detail";
import { AboutPage } from "@/pages/about";
import { NotFoundPage } from "@/pages/not-found";
import "./index.css";

initTheme();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1 },
  },
});

// Auth guard: in dev mode always allow; in prod cookie-based session
// is validated by the AuthProvider on mount.
function requireAuth() {
  if (import.meta.env.DEV) return;
  // Cookie-based auth: HttpOnly cookie can't be checked from JS.
  // AuthProvider validates the session on mount via /auth/me.
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
const authCallbackRoute = createRoute({
  getParentRoute: () => publicRootRoute,
  path: "/auth/callback",
  component: AuthCallbackPage,
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
const adminAuthRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/admin/auth",
  component: AdminAuthPage,
});
const aboutRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/about",
  component: AboutPage,
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
  authCallbackRoute,
  appLayoutRoute.addChildren([
    indexRoute,
    ticketDetailRoute,
    slaRoute,
    adminRoute,
    adminZammadRoute,
    adminIntegrationsRoute,
    adminCustomersRoute,
    adminRulesRoute,
    adminAuthRoute,
    aboutRoute,
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
