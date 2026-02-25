package zammad_test

import (
	"context"
	"testing"

	"github.com/wisbric/ticketowl/internal/zammad"
)

func TestGetTicket(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddTicket(zammad.Ticket{
		ID:     42,
		Number: "10042",
		Title:  "Printer on fire",
		State:  "open",
	})

	ticket, err := client.GetTicket(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetTicket: unexpected error: %v", err)
	}
	if ticket.ID != 42 {
		t.Errorf("ID = %d, want 42", ticket.ID)
	}
	if ticket.Title != "Printer on fire" {
		t.Errorf("Title = %q, want %q", ticket.Title, "Printer on fire")
	}

	m.RequireCall(t, "GET", "/api/v1/tickets/42")
}

func TestGetTicket_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	_, err := client.GetTicket(context.Background(), 999)
	if err == nil {
		t.Fatal("GetTicket: expected error for missing ticket")
	}
	if !zammad.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestListTickets(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddTicket(zammad.Ticket{ID: 1, Title: "First"})
	m.AddTicket(zammad.Ticket{ID: 2, Title: "Second"})
	m.AddTicket(zammad.Ticket{ID: 3, Title: "Third"})

	tickets, err := client.ListTickets(context.Background(), zammad.ListTicketsOptions{})
	if err != nil {
		t.Fatalf("ListTickets: unexpected error: %v", err)
	}
	if len(tickets) != 3 {
		t.Errorf("len(tickets) = %d, want 3", len(tickets))
	}
}

func TestCreateTicket(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	ticket, err := client.CreateTicket(context.Background(), zammad.TicketCreateRequest{
		Title:      "New ticket",
		GroupID:    1,
		CustomerID: 10,
	})
	if err != nil {
		t.Fatalf("CreateTicket: unexpected error: %v", err)
	}
	if ticket.Title != "New ticket" {
		t.Errorf("Title = %q, want %q", ticket.Title, "New ticket")
	}
	if ticket.ID == 0 {
		t.Error("expected non-zero ticket ID")
	}
	if ticket.StateID != 1 {
		t.Errorf("StateID = %d, want 1 (default new)", ticket.StateID)
	}

	m.RequireCall(t, "POST", "/api/v1/tickets")
}

func TestUpdateTicket(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddTicket(zammad.Ticket{ID: 50, Title: "Update me", StateID: 1, PriorityID: 2})

	newState := 5
	ticket, err := client.UpdateTicket(context.Background(), 50, zammad.TicketUpdateRequest{
		StateID: &newState,
	})
	if err != nil {
		t.Fatalf("UpdateTicket: unexpected error: %v", err)
	}
	if ticket.StateID != 5 {
		t.Errorf("StateID = %d, want 5", ticket.StateID)
	}

	m.RequireCall(t, "PUT", "/api/v1/tickets/50")
}

func TestUpdateTicket_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	newState := 2
	_, err := client.UpdateTicket(context.Background(), 999, zammad.TicketUpdateRequest{
		StateID: &newState,
	})
	if err == nil {
		t.Fatal("UpdateTicket: expected error for missing ticket")
	}
	if !zammad.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestSearchTickets(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddTicket(zammad.Ticket{ID: 10, Title: "Searchable ticket"})

	tickets, err := client.SearchTickets(context.Background(), "Searchable", zammad.ListTicketsOptions{})
	if err != nil {
		t.Fatalf("SearchTickets: unexpected error: %v", err)
	}
	if len(tickets) == 0 {
		t.Error("expected at least one search result")
	}

	m.RequireCall(t, "GET", "/api/v1/tickets/search")
}

func TestListArticles(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddArticle(42, zammad.Article{Body: "First comment"})
	m.AddArticle(42, zammad.Article{Body: "Second comment"})

	articles, err := client.ListArticles(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListArticles: unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("len(articles) = %d, want 2", len(articles))
	}
	if articles[0].Body != "First comment" {
		t.Errorf("articles[0].Body = %q, want %q", articles[0].Body, "First comment")
	}

	m.RequireCall(t, "GET", "/api/v1/ticket_articles/by_ticket/42")
}

func TestListArticles_Empty(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	articles, err := client.ListArticles(context.Background(), 999)
	if err != nil {
		t.Fatalf("ListArticles: unexpected error: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("len(articles) = %d, want 0", len(articles))
	}
}

func TestCreateArticle(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	article, err := client.CreateArticle(context.Background(), zammad.ArticleCreate{
		TicketID:    42,
		Body:        "A new note",
		ContentType: "text/plain",
		Internal:    true,
	})
	if err != nil {
		t.Fatalf("CreateArticle: unexpected error: %v", err)
	}
	if article.Body != "A new note" {
		t.Errorf("Body = %q, want %q", article.Body, "A new note")
	}
	if article.TicketID != 42 {
		t.Errorf("TicketID = %d, want 42", article.TicketID)
	}
	if !article.Internal {
		t.Error("expected Internal=true")
	}

	m.RequireCall(t, "POST", "/api/v1/ticket_articles")
}

func TestGetUser(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddUser(zammad.User{
		ID:        5,
		Firstname: "Jane",
		Lastname:  "Doe",
		Email:     "jane@example.com",
		Active:    true,
	})

	user, err := client.GetUser(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetUser: unexpected error: %v", err)
	}
	if user.ID != 5 {
		t.Errorf("ID = %d, want 5", user.ID)
	}
	if user.Firstname != "Jane" {
		t.Errorf("Firstname = %q, want %q", user.Firstname, "Jane")
	}
	if user.Email != "jane@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "jane@example.com")
	}
}

func TestGetUser_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	_, err := client.GetUser(context.Background(), 999)
	if err == nil {
		t.Fatal("GetUser: expected error for missing user")
	}
	if !zammad.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestGetCurrentUser(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser: unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("ID = %d, want 1", user.ID)
	}
	if user.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "admin@example.com")
	}

	m.RequireCall(t, "GET", "/api/v1/users/me")
}

func TestGetOrganisation(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddOrganisation(zammad.Organisation{
		ID:     3,
		Name:   "Acme Corp",
		Active: true,
	})

	org, err := client.GetOrganisation(context.Background(), 3)
	if err != nil {
		t.Fatalf("GetOrganisation: unexpected error: %v", err)
	}
	if org.ID != 3 {
		t.Errorf("ID = %d, want 3", org.ID)
	}
	if org.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", org.Name, "Acme Corp")
	}
}

func TestGetOrganisation_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	_, err := client.GetOrganisation(context.Background(), 999)
	if err == nil {
		t.Fatal("GetOrganisation: expected error for missing org")
	}
	if !zammad.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestListStates(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	states, err := client.ListStates(context.Background())
	if err != nil {
		t.Fatalf("ListStates: unexpected error: %v", err)
	}
	if len(states) != 4 {
		t.Errorf("len(states) = %d, want 4", len(states))
	}

	m.RequireCall(t, "GET", "/api/v1/ticket_states")
}

func TestListPriorities(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	priorities, err := client.ListPriorities(context.Background())
	if err != nil {
		t.Fatalf("ListPriorities: unexpected error: %v", err)
	}
	if len(priorities) != 3 {
		t.Errorf("len(priorities) = %d, want 3", len(priorities))
	}

	m.RequireCall(t, "GET", "/api/v1/ticket_priorities")
}

func TestUnauthorized(t *testing.T) {
	m := NewMockServer(t)

	// Create a client with no token — the mock server rejects requests
	// without "Token token=..." in the Authorization header.
	client := zammad.New(m.URL(), "")

	_, err := client.GetCurrentUser(context.Background())
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestCallCount(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddTicket(zammad.Ticket{ID: 1, Title: "T1"})

	_, _ = client.GetTicket(context.Background(), 1)
	_, _ = client.GetTicket(context.Background(), 1)

	if got := m.CallCount("GET", "/api/v1/tickets/1"); got != 2 {
		t.Errorf("CallCount = %d, want 2", got)
	}
}

func TestRequireNoCall(t *testing.T) {
	m := NewMockServer(t)
	// No calls made, so this should pass.
	m.RequireNoCall(t, "DELETE", "/api/v1/tickets/1")
}

func TestValidateWebhookSignature(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		secret  string
		sig     string
		wantErr bool
	}{
		{
			name:    "valid signature",
			body:    []byte(`{"event":"ticket.update"}`),
			secret:  "test-secret",
			sig:     zammad.ComputeWebhookSignature([]byte(`{"event":"ticket.update"}`), "test-secret"),
			wantErr: false,
		},
		{
			name:    "wrong secret",
			body:    []byte(`{"event":"ticket.update"}`),
			secret:  "test-secret",
			sig:     zammad.ComputeWebhookSignature([]byte(`{"event":"ticket.update"}`), "wrong-secret"),
			wantErr: true,
		},
		{
			name:    "tampered body",
			body:    []byte(`{"event":"ticket.delete"}`),
			secret:  "test-secret",
			sig:     zammad.ComputeWebhookSignature([]byte(`{"event":"ticket.update"}`), "test-secret"),
			wantErr: true,
		},
		{
			name:    "invalid format — missing prefix",
			body:    []byte(`{}`),
			secret:  "s",
			sig:     "deadbeef",
			wantErr: true,
		},
		{
			name:    "invalid format — too short",
			body:    []byte(`{}`),
			secret:  "s",
			sig:     "sha",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := zammad.ValidateWebhookSignature(tt.body, tt.secret, tt.sig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookSignature() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestComputeWebhookSignature(t *testing.T) {
	body := []byte(`{"ticket":{"id":1}}`)
	secret := "my-secret"

	sig := zammad.ComputeWebhookSignature(body, secret)
	if len(sig) < 6 {
		t.Fatalf("signature too short: %q", sig)
	}
	if sig[:5] != "sha1=" {
		t.Errorf("signature missing sha1= prefix: %q", sig)
	}

	// Round-trip: validate what we computed.
	if err := zammad.ValidateWebhookSignature(body, secret, sig); err != nil {
		t.Errorf("computed signature failed validation: %v", err)
	}
}

func TestIsNotFound(t *testing.T) {
	if zammad.IsNotFound(nil) {
		t.Error("IsNotFound(nil) = true, want false")
	}

	err := &zammad.ZammadError{StatusCode: 404, Message: "not found"}
	if !zammad.IsNotFound(err) {
		t.Error("IsNotFound(404) = false, want true")
	}

	err = &zammad.ZammadError{StatusCode: 500, Message: "server error"}
	if zammad.IsNotFound(err) {
		t.Error("IsNotFound(500) = true, want false")
	}
}

func TestIsUnauthorised(t *testing.T) {
	if zammad.IsUnauthorised(nil) {
		t.Error("IsUnauthorised(nil) = true, want false")
	}

	for _, code := range []int{401, 403} {
		err := &zammad.ZammadError{StatusCode: code, Message: "denied"}
		if !zammad.IsUnauthorised(err) {
			t.Errorf("IsUnauthorised(%d) = false, want true", code)
		}
	}

	err := &zammad.ZammadError{StatusCode: 404, Message: "not found"}
	if zammad.IsUnauthorised(err) {
		t.Error("IsUnauthorised(404) = true, want false")
	}
}
