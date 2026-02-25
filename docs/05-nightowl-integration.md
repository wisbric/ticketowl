# 05 — NightOwl Integration

## Overview

NightOwl is the incident and on-call platform. TicketOwl integrates in both directions: it calls NightOwl's API to fetch incident data and fire alerts, and it receives webhooks from NightOwl when incidents are created or resolved.

---

## Outbound Client — `internal/nightowl/`

```
internal/nightowl/
  client.go       — base client (same pattern as internal/zammad/)
  incidents.go    — GetIncident, ListIncidents
  oncall.go       — GetOnCallForService
  alerts.go       — CreateAlert (SLA breach escalation)
  mock_test.go    — httptest mock for unit tests
```

The client uses the tenant's NightOwl API key stored in `integration_keys` (service = `nightowl`).

### Methods

```go
func (c *Client) GetIncident(ctx, incidentID string) (*Incident, error)
func (c *Client) GetOnCallForService(ctx, service string) (*OnCallInfo, error)
func (c *Client) CreateAlert(ctx, CreateAlertRequest) (*Alert, error)

type Incident struct {
    ID       string
    Slug     string
    Summary  string
    Severity string
    Status   string    // "open" | "acknowledged" | "resolved"
    Service  string
    Tags     []string
    CreatedAt time.Time
    ResolvedAt *time.Time
}

type OnCallInfo struct {
    UserName  string
    UserEmail string
    ShiftEnd  time.Time
}

type CreateAlertRequest struct {
    Name        string
    Summary     string
    Severity    string
    Labels      map[string]string
    Annotations map[string]string
}
```

---

## Inbound Webhook — `POST /api/v1/webhooks/nightowl`

NightOwl posts events when incidents are created or resolved.

```go
type NightOwlEvent struct {
    Event    string    // "incident.created" | "incident.resolved"
    Incident Incident
}
```

### incident.created flow

```
1. Validate X-API-Key (NightOwl's API key for this tenant)
2. Push to Redis stream
Worker picks up:
3. Evaluate all enabled auto_ticket_rules against the incident
4. For each matching rule:
   a. Build ticket title from rule.title_template + incident data
   b. Create Zammad ticket (title, group, priority from rule defaults)
   c. UpsertTicketMeta, assign SLA policy
   d. Create incident_links row linking the new ticket to the incident
```

### incident.resolved flow

```
1. Validate, push to Redis stream
Worker picks up:
2. Find all incident_links for this incident_id
3. For each linked ticket:
   a. Fetch current Zammad ticket state
   b. If not already closed: update Zammad ticket to status "closed"
   c. Optionally surface post-mortem prompt (no auto-creation — agent decides)
```

---

## SLA Breach Escalation

When the SLA worker detects a new breach, it calls `notification.Service.AlertSLABreach()` which calls `nightowl.Client.CreateAlert()` with:

```json
{
  "name": "SLA Breach",
  "summary": "Ticket #1234 has breached SLA (resolution, critical priority)",
  "severity": "high",
  "labels": {
    "source": "ticketowl",
    "ticket_id": "1234",
    "ticket_number": "00042",
    "sla_type": "resolution",
    "priority": "critical"
  }
}
```

This creates an incident in NightOwl and pages the on-call engineer. TicketOwl then stores `first_breach_alerted_at` in `sla_states` to prevent duplicate pages.

---

## On-Call Display

On the ticket detail page, TicketOwl fetches the current on-call for the ticket's service (derived from the linked incident's `service` field). If no incident is linked, the on-call widget is hidden.

The response is fetched fresh on every ticket detail page load — not cached — to always show the current on-call person.
