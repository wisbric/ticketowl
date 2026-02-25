# 06 — BookOwl Integration

## Overview

BookOwl is the knowledge base platform. TicketOwl integrates with it to surface relevant runbooks and articles on ticket detail pages, and to create post-mortem drafts when incidents are resolved.

---

## Outbound Client — `internal/bookowl/`

```
internal/bookowl/
  client.go       — base client (same pattern as internal/zammad/)
  search.go       — SearchArticles
  articles.go     — GetArticle
  postmortems.go  — CreatePostMortem
  mock_test.go    — httptest mock for unit tests
```

The client uses the tenant's BookOwl API key stored in `integration_keys` (service = `bookowl`).

### Methods

```go
func (c *Client) SearchArticles(ctx, SearchOptions) ([]ArticleSummary, error)
func (c *Client) GetArticle(ctx, articleID string) (*Article, error)
func (c *Client) CreatePostMortem(ctx, CreatePostMortemRequest) (*PostMortem, error)

type SearchOptions struct {
    Query string
    Tags  []string
    Limit int
}

type ArticleSummary struct {
    ID    string
    Slug  string
    Title string
    Excerpt string
    Tags  []string
    URL   string
}

type CreatePostMortemRequest struct {
    Title       string
    TicketID    string   // TicketOwl reference
    TicketNumber string
    IncidentIDs []string
    Summary     string   // pre-filled from incident data
    Tags        []string
}

type PostMortem struct {
    ID  string
    URL string
}
```

---

## Runbook Suggestions

`GET /api/v1/tickets/{id}/suggestions` calls `bookowl.Client.SearchArticles()` with:
- `query`: the ticket title
- `tags`: the Zammad ticket's tags

Returns up to 5 article summaries. Results are not cached — fetched fresh each time the ticket detail loads.

If BookOwl is unreachable, return an empty suggestions list with a `503` and log the error. Do not fail the ticket detail request.

---

## Post-Mortem Creation

`POST /api/v1/tickets/{id}/postmortem` triggers post-mortem creation when an agent decides it's appropriate (typically after incident resolution).

Flow:
1. Load the ticket's linked incidents from `incident_links`
2. Fetch incident details from NightOwl for each linked incident
3. Build a `CreatePostMortemRequest` pre-filled with ticket and incident data
4. Call `bookowl.Client.CreatePostMortem()`
5. Store the returned URL in `postmortem_links`
6. Return the post-mortem URL to the frontend

If a post-mortem already exists for this ticket (row in `postmortem_links`), return 409 with the existing URL.

---

## Article Linking

Agents can search for and attach specific BookOwl articles to a ticket via `POST /api/v1/tickets/{id}/links/article`. This:
1. Calls `bookowl.Client.GetArticle()` to validate the article exists
2. Stores an `article_links` row with a snapshot of the title
3. Returns the link

Linked articles are shown both in the agent view and in the customer portal.
