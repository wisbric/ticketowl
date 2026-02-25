package nightowl_test

import (
	"context"
	"testing"
	"time"

	"github.com/wisbric/ticketowl/internal/nightowl"
)

func TestGetIncident(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddIncident(nightowl.Incident{
		ID:       "inc-001",
		Slug:     "api-outage",
		Summary:  "API is down",
		Severity: "critical",
		Status:   "open",
		Service:  "api-gateway",
	})

	incident, err := client.GetIncident(context.Background(), "inc-001")
	if err != nil {
		t.Fatalf("GetIncident: unexpected error: %v", err)
	}
	if incident.ID != "inc-001" {
		t.Errorf("ID = %q, want %q", incident.ID, "inc-001")
	}
	if incident.Summary != "API is down" {
		t.Errorf("Summary = %q, want %q", incident.Summary, "API is down")
	}
	if incident.Service != "api-gateway" {
		t.Errorf("Service = %q, want %q", incident.Service, "api-gateway")
	}

	m.RequireCall(t, "GET", "/api/v1/incidents/inc-001")
}

func TestGetIncident_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	_, err := client.GetIncident(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("GetIncident: expected error for missing incident")
	}
	if !nightowl.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestListIncidents(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddIncident(nightowl.Incident{ID: "inc-001", Status: "open"})
	m.AddIncident(nightowl.Incident{ID: "inc-002", Status: "resolved"})

	incidents, err := client.ListIncidents(context.Background(), "")
	if err != nil {
		t.Fatalf("ListIncidents: unexpected error: %v", err)
	}
	if len(incidents) != 2 {
		t.Errorf("len(incidents) = %d, want 2", len(incidents))
	}

	m.RequireCall(t, "GET", "/api/v1/incidents")
}

func TestGetOnCallForService(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	shiftEnd := time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC)
	m.AddOnCall("api-gateway", nightowl.OnCallInfo{
		UserName:  "Jane Doe",
		UserEmail: "jane@example.com",
		ShiftEnd:  shiftEnd,
	})

	info, err := client.GetOnCallForService(context.Background(), "api-gateway")
	if err != nil {
		t.Fatalf("GetOnCallForService: unexpected error: %v", err)
	}
	if info.UserName != "Jane Doe" {
		t.Errorf("UserName = %q, want %q", info.UserName, "Jane Doe")
	}
	if info.UserEmail != "jane@example.com" {
		t.Errorf("UserEmail = %q, want %q", info.UserEmail, "jane@example.com")
	}

	m.RequireCall(t, "GET", "/api/v1/oncall/api-gateway")
}

func TestGetOnCallForService_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	_, err := client.GetOnCallForService(context.Background(), "no-service")
	if err == nil {
		t.Fatal("GetOnCallForService: expected error for unknown service")
	}
	if !nightowl.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestCreateAlert(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	alert, err := client.CreateAlert(context.Background(), nightowl.CreateAlertRequest{
		Name:     "SLA Breach",
		Summary:  "Ticket #1234 breached resolution SLA",
		Severity: "high",
		Labels: map[string]string{
			"source":    "ticketowl",
			"ticket_id": "1234",
		},
	})
	if err != nil {
		t.Fatalf("CreateAlert: unexpected error: %v", err)
	}
	if alert.ID == "" {
		t.Error("expected non-empty alert ID")
	}
	if alert.Name != "SLA Breach" {
		t.Errorf("Name = %q, want %q", alert.Name, "SLA Breach")
	}
	if alert.Status != "firing" {
		t.Errorf("Status = %q, want %q", alert.Status, "firing")
	}

	m.RequireCall(t, "POST", "/api/v1/alerts")

	alerts := m.Alerts()
	if len(alerts) != 1 {
		t.Errorf("len(alerts) = %d, want 1", len(alerts))
	}
}

func TestUnauthorized(t *testing.T) {
	m := NewMockServer(t)
	client := nightowl.New(m.URL(), "")

	_, err := client.GetIncident(context.Background(), "inc-001")
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestCallCount(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddIncident(nightowl.Incident{ID: "inc-001"})

	_, _ = client.GetIncident(context.Background(), "inc-001")
	_, _ = client.GetIncident(context.Background(), "inc-001")
	_, _ = client.GetIncident(context.Background(), "inc-001")

	if got := m.CallCount("GET", "/api/v1/incidents/inc-001"); got != 3 {
		t.Errorf("CallCount = %d, want 3", got)
	}
}

func TestRequireNoCall(t *testing.T) {
	m := NewMockServer(t)
	m.RequireNoCall(t, "DELETE", "/api/v1/incidents/inc-001")
}

func TestIsNotFound(t *testing.T) {
	if nightowl.IsNotFound(nil) {
		t.Error("IsNotFound(nil) = true, want false")
	}

	err := &nightowl.APIError{StatusCode: 404, Message: "not found"}
	if !nightowl.IsNotFound(err) {
		t.Error("IsNotFound(404) = false, want true")
	}

	err = &nightowl.APIError{StatusCode: 500, Message: "server error"}
	if nightowl.IsNotFound(err) {
		t.Error("IsNotFound(500) = true, want false")
	}
}

func TestIsUnauthorised(t *testing.T) {
	if nightowl.IsUnauthorised(nil) {
		t.Error("IsUnauthorised(nil) = true, want false")
	}

	for _, code := range []int{401, 403} {
		err := &nightowl.APIError{StatusCode: code, Message: "denied"}
		if !nightowl.IsUnauthorised(err) {
			t.Errorf("IsUnauthorised(%d) = false, want true", code)
		}
	}

	err := &nightowl.APIError{StatusCode: 404, Message: "not found"}
	if nightowl.IsUnauthorised(err) {
		t.Error("IsUnauthorised(404) = true, want false")
	}
}
