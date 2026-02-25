package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/wisbric/ticketowl/internal/nightowl"
)

// NightOwlAlerter defines the NightOwl operations the notification service needs.
type NightOwlAlerter interface {
	CreateAlert(ctx context.Context, req nightowl.CreateAlertRequest) (*nightowl.Alert, error)
}

// Service handles sending notifications for SLA breaches.
type Service struct {
	nightowl NightOwlAlerter
	logger   *slog.Logger
}

// NewService creates a notification Service.
func NewService(nightowl NightOwlAlerter, logger *slog.Logger) *Service {
	return &Service{nightowl: nightowl, logger: logger}
}

// BreachInfo describes an SLA breach to be escalated.
type BreachInfo struct {
	TicketZammadID int
	TicketNumber   string
	SLAType        string // "response" | "resolution"
	Priority       string
}

// AlertSLABreach sends an SLA breach alert to NightOwl.
func (s *Service) AlertSLABreach(ctx context.Context, info BreachInfo) error {
	req := nightowl.CreateAlertRequest{
		Name:     "SLA Breach",
		Summary:  fmt.Sprintf("Ticket #%s has breached SLA (%s, %s priority)", info.TicketNumber, info.SLAType, info.Priority),
		Severity: "high",
		Labels: map[string]string{
			"source":        "ticketowl",
			"ticket_id":     strconv.Itoa(info.TicketZammadID),
			"ticket_number": info.TicketNumber,
			"sla_type":      info.SLAType,
			"priority":      info.Priority,
		},
	}

	alert, err := s.nightowl.CreateAlert(ctx, req)
	if err != nil {
		return fmt.Errorf("creating NightOwl alert for SLA breach: %w", err)
	}

	s.logger.Info("SLA breach alert sent",
		"alert_id", alert.ID,
		"ticket_number", info.TicketNumber,
		"sla_type", info.SLAType,
	)

	return nil
}
