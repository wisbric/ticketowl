# TicketOwl Web

The TicketOwl frontend — a React 19 SPA that serves as the unified support portal for internal agents and external customers.

## Tech Stack

- **React 19** + TypeScript 5.9
- **Vite 7** (dev server + build)
- **Tailwind CSS 4** with NightOwl design tokens (dark mode default)
- **TanStack Router** for file-based routing with type-safe params
- **TanStack Query** for all API data fetching and mutations
- **lucide-react** for icons

## Development

```bash
# From repo root — start backend + dependencies first
docker compose up -d
make seed
make api      # API on :8082

# Start the frontend dev server
cd web
npm install
npm run dev   # http://localhost:3002
```

The Vite dev server proxies `/api` requests to `localhost:8082` (the Go API).

In dev mode, authentication is automatic — the auth context uses the dev API key `to_dev_seed_key_do_not_use_in_production`.

## Build

```bash
npm run build   # TypeScript check + Vite production build → dist/
npm run preview # Preview the production build locally
```

## Project Structure

```
src/
├── components/
│   ├── layout/           # AppLayout, Sidebar, PortalLayout
│   ├── tickets/          # SLA badges, thread view, modals, suggestions
│   └── ui/               # Shared UI primitives (button, card, table, dialog, etc.)
├── contexts/
│   └── auth-context.tsx  # Auth provider (dev auto-auth / prod OIDC)
├── hooks/
│   ├── use-theme.ts      # Dark/light mode toggle
│   └── use-title.ts      # Page title helper
├── lib/
│   ├── api.ts            # Typed fetch wrapper with auth headers
│   └── utils.ts          # cn(), formatRelativeTime(), SLA/priority helpers
├── pages/
│   ├── ticket-list.tsx   # The Perch — ticket queue with filters
│   ├── ticket-detail.tsx # Full ticket view with thread, SLA, links, notes
│   ├── portal-*.tsx      # Customer portal views
│   ├── admin-*.tsx       # Admin configuration pages
│   └── admin-sla.tsx     # The Watch — SLA policy management
├── types/
│   └── api.ts            # TypeScript interfaces matching Go backend structs
├── index.css             # NightOwl design tokens (dark/light CSS variables)
└── main.tsx              # Router tree, QueryClient, AuthProvider
```

## Pages

| Route | Page | Description |
|-------|------|-------------|
| `/` | The Perch | Ticket list with status/priority filters |
| `/tickets/:id` | Ticket Detail | Thread, internal notes, SLA timer, linked incidents/articles |
| `/sla` | The Watch | SLA policy management |
| `/admin` | Admin | Configuration landing page |
| `/admin/zammad` | Zammad Config | Zammad instance URL, API token, webhook secret |
| `/admin/integrations` | Integrations | NightOwl + BookOwl API keys and URLs |
| `/admin/customers` | Customers | Customer organization management |
| `/admin/rules` | Rules | Auto-ticket rule management |
| `/portal/tickets` | Portal List | Customer-facing ticket list |
| `/portal/tickets/:id` | Portal Detail | Customer-facing ticket detail with reply |

## Design System

TicketOwl uses the NightOwl design system with semantic CSS custom properties:

- **Dark mode** is the default (`:root` defines dark tokens)
- **Light mode** is toggled via `.light` class on `<html>`
- Primary: `#0A2E24`, Accent: `#00e5a0`
- SLA state colors: green (on track), amber (warning), red (breached), blue (paused)
- Theme preference is stored in `localStorage` under `ticketowl_theme`
