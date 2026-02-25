# 09 — Branding

## Identity

The product is called **TicketOwl**. It is part of the NightOwl platform family by Wisbric.

---

## Design System

TicketOwl uses the **NightOwl design system** — same Tailwind tokens, same shadcn/ui components, same dark-first approach. Do not introduce new design primitives.

### Colour Tokens

```
Background primary:   #0A2E24   (dark green)
Accent:               #00e5a0   (bright teal)
Surface:              #0D1F1A
Border:               #1A3A2E
Text primary:         #E8F5F0
Text muted:           #6B9E8A
```

### SLA State Colours (follow NightOwl severity palette)

```
on_track:  accent   #00e5a0
warning:   yellow   #f59e0b
breached:  critical #ef4444
met:       muted    #6B9E8A
paused:    info     #3b82f6
```

### Severity Colours (for linked incidents)

```
critical:  #ef4444
high:      #f97316
medium:    #f59e0b
low:       #6b7280
```

---

## Owl-Themed Naming (UI only)

These flavor names appear in the UI. Use plain English in API routes, database columns, and code.

| Technical term     | UI label      |
|--------------------|---------------|
| Ticket             | Case          |
| Queue / ticket list | The Perch    |
| SLA status widget  | The Watch     |

---

## Layout

- **Sidebar navigation** (left): NightOwl/BookOwl/TicketOwl product switcher at top, then nav items
- **Header**: product name + tenant name + user menu
- **Main content area**: full width, dark background
- **Footer**: "A Wisbric product"

---

## Dark Mode

Dark mode is the **default**. A toggle in the user menu allows switching to light mode. All components must look correct in dark mode before light mode is considered.

---

## Logo

Use the shared owl logo asset from NightOwl. No separate TicketOwl logo in v1 — the product switcher differentiates products by name.
