package notification_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/notification"
)

type mockAlerter struct {
	alerts []nightowl.CreateAlertRequest
}

func (m *mockAlerter) CreateAlert(_ context.Context, req nightowl.CreateAlertRequest) (*nightowl.Alert, error) {
	m.alerts = append(m.alerts, req)
	return &nightowl.Alert{
		ID:       "alert-001",
		Name:     req.Name,
		Severity: req.Severity,
		Status:   "firing",
	}, nil
}

func TestAlertSLABreach(t *testing.T) {
	alerter := &mockAlerter{}
	svc := notification.NewService(alerter, slog.Default())

	err := svc.AlertSLABreach(context.Background(), notification.BreachInfo{
		TicketZammadID: 42,
		TicketNumber:   "T42",
		SLAType:        "resolution",
		Priority:       "critical",
	})
	if err != nil {
		t.Fatalf("AlertSLABreach: unexpected error: %v", err)
	}

	if len(alerter.alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerter.alerts))
	}

	alert := alerter.alerts[0]
	if alert.Name != "SLA Breach" {
		t.Errorf("alert name = %q, want %q", alert.Name, "SLA Breach")
	}
	if alert.Severity != "high" {
		t.Errorf("severity = %q, want high", alert.Severity)
	}
	if alert.Labels["source"] != "ticketowl" {
		t.Errorf("label source = %q, want ticketowl", alert.Labels["source"])
	}
	if alert.Labels["ticket_id"] != "42" {
		t.Errorf("label ticket_id = %q, want 42", alert.Labels["ticket_id"])
	}
	if alert.Labels["ticket_number"] != "T42" {
		t.Errorf("label ticket_number = %q, want T42", alert.Labels["ticket_number"])
	}
	if alert.Labels["sla_type"] != "resolution" {
		t.Errorf("label sla_type = %q, want resolution", alert.Labels["sla_type"])
	}
	if alert.Labels["priority"] != "critical" {
		t.Errorf("label priority = %q, want critical", alert.Labels["priority"])
	}
}

func TestAlertSLABreach_SummaryFormat(t *testing.T) {
	alerter := &mockAlerter{}
	svc := notification.NewService(alerter, slog.Default())

	_ = svc.AlertSLABreach(context.Background(), notification.BreachInfo{
		TicketZammadID: 99,
		TicketNumber:   "00042",
		SLAType:        "response",
		Priority:       "high",
	})

	want := "Ticket #00042 has breached SLA (response, high priority)"
	if alerter.alerts[0].Summary != want {
		t.Errorf("summary = %q, want %q", alerter.alerts[0].Summary, want)
	}
}
