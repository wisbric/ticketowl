# 04 — Zammad Integration

## Overview

Zammad is the authoritative ticketing engine. TicketOwl communicates with it via the Zammad REST API using HTTP Token auth. There is no official Go SDK — maintain a typed client in `internal/zammad/`.

Each tenant stores its own Zammad URL and API token, loaded from the `zammad_config` table at request time.

---

## Client Package — `internal/zammad/`

```
internal/zammad/
  client.go      — Client struct, constructor, base HTTP helpers, retry logic
  tickets.go     — Ticket CRUD
  comments.go    — Article (comment) operations
  users.go       — User lookup
  orgs.go        — Organisation lookup
  states.go      — State and priority lookups (Redis-cached, TTL 1h)
  webhook.go     — Webhook payload types and HMAC-SHA1 validation
  errors.go      — ZammadError type, IsNotFound, IsUnauthorised helpers
  mock_test.go   — httptest-based mock server for unit tests
```

### Client Construction

```go
type Client struct { /* baseURL, token, httpClient, logger, tracer */ }

func New(baseURL, token string, opts ...Option) *Client
```

The client is constructed per-request in the service layer using the tenant's stored config. It is not a singleton.

### Retry Policy

Retry non-2xx responses (except 4xx) with exponential backoff:
- Max attempts: 3
- Base delay: 250ms, multiplier: 2x, jitter: ±20%
- Do **not** retry: 400, 401, 403, 404, 422

---

## Ticket API

### Types

```go
type Ticket struct {
    ID             int
    Number         string
    Title          string
    StateID        int
    State          string   // expanded via ?expand=true
    PriorityID     int
    Priority       string
    GroupID        int
    Group          string
    OwnerID        int
    Owner          string
    CustomerID     int
    OrganisationID int
    Tags           []string
    CreatedAt      time.Time
    UpdatedAt      time.Time
    CloseAt        *time.Time
}

type TicketCreateRequest struct {
    Title      string
    GroupID    int
    CustomerID int
    StateID    int            // optional
    PriorityID int            // optional
    Article    *ArticleCreate // optional first article
}

type TicketUpdateRequest struct {
    StateID    *int
    PriorityID *int
    OwnerID    *int
    GroupID    *int
}
```

### Methods

```go
func (c *Client) ListTickets(ctx, ListTicketsOptions) ([]Ticket, error)
func (c *Client) GetTicket(ctx, id int) (*Ticket, error)
func (c *Client) CreateTicket(ctx, TicketCreateRequest) (*Ticket, error)
func (c *Client) UpdateTicket(ctx, id int, TicketUpdateRequest) (*Ticket, error)
func (c *Client) SearchTickets(ctx, query string, ListTicketsOptions) ([]Ticket, error)

type ListTicketsOptions struct {
    Page     int
    PerPage  int
    StateIDs []int
    GroupIDs []int
    OrgID    *int   // for customer portal scoping
}
```

---

## Article (Comment) API

In Zammad, comments are "articles". `internal: true` = internal note (agent only). `internal: false` = public reply (customer visible).

```go
type Article struct {
    ID          int
    TicketID    int
    Type        string     // "note" | "web" | "email"
    Sender      string     // "Customer" | "Agent" | "System"
    Body        string
    ContentType string     // "text/html" | "text/plain"
    Internal    bool
    CreatedBy   string
    CreatedAt   time.Time
}

type ArticleCreate struct {
    TicketID    int
    TypeID      int    // 10 = note, 11 = web
    Body        string
    ContentType string
    Internal    bool
    SenderID    int    // 1=Customer, 2=Agent, 3=System
}

func (c *Client) ListArticles(ctx, ticketID int) ([]Article, error)
func (c *Client) CreateArticle(ctx, ArticleCreate) (*Article, error)
```

---

## State & Priority Lookups

Zammad uses integer IDs for states and priorities. Cache these per-tenant in Redis (TTL 1h).

```go
func (c *Client) ListStates(ctx) ([]TicketState, error)
func (c *Client) ListPriorities(ctx) ([]TicketPriority, error)
```

Redis keys: `ticketowl:{tenant}:zammad:states` and `ticketowl:{tenant}:zammad:priorities`.

**Default Zammad state IDs** (confirm per instance, but these are typical):

| State            | ID |
|------------------|----|
| new              | 1  |
| open             | 2  |
| pending reminder | 3  |
| closed           | 5  |

---

## Webhook Integration

### Setup

TicketOwl registers a Zammad trigger that POSTs to `{ticketowl_url}/api/v1/webhooks/zammad` on:
- Ticket created
- Ticket updated (status, priority, assignee)
- Article created

Must be configured in Zammad admin during tenant provisioning.

### Signature Validation

Validate the `X-Hub-Signature` header (HMAC-SHA1) on every inbound webhook before processing.

```go
type WebhookPayload struct {
    Event   string         // "ticket.create" | "ticket.update" | "article.create"
    Ticket  *WebhookTicket
    Article *Article       // only on article.create
}

type WebhookTicket struct {
    ID        int
    Number    string
    StateID   int
    State     string
    Priority  string
    UpdatedAt time.Time
}
```

### Webhook Handler Flow

```
POST /api/v1/webhooks/zammad
  1. Validate HMAC-SHA1 signature
  2. Parse payload
  3. Push event to Redis stream: XADD ticketowl:{tenant}:events * ...
  4. Return 200 immediately

Worker event processor (XREAD):
  ticket.create  → UpsertTicketMeta, assign SLA policy, init SLA state
  ticket.update  → check pause status change, recompute SLA
  article.create → reserved for future notification hooks
```

---

## Connection Test

`POST /api/v1/admin/config/zammad/test` calls `GET /api/v1/users/me` and returns:

```json
{ "ok": true, "zammad_version": "6.3.0", "agent_name": "...", "agent_email": "..." }
```

or:

```json
{ "ok": false, "error": "connection refused" }
```

---

## Mock Server for Tests

`internal/zammad/mock_test.go` provides an `httptest.Server`-backed mock:

```go
type MockServer struct {
    // in-memory tickets and articles
    // recorded calls for assertion
}

func NewMockServer(t *testing.T) *MockServer
func (m *MockServer) Client() *Client
func (m *MockServer) AddTicket(Ticket)
func (m *MockServer) AddArticle(ticketID int, Article)
func (m *MockServer) RequireCall(t, method, path string)
func (m *MockServer) RequireNoCall(t, method, path string)
```

All ticket service unit tests use this mock — never hit a real Zammad in tests.
