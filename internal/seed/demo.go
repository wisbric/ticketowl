package seed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/core/pkg/tenant"
)

// RunDemo drops and recreates the acme tenant with demo data.
// This is destructive — intended for UI development only.
func RunDemo(ctx context.Context, db *pgxpool.Pool, databaseURL, migrationsDir string, logger *slog.Logger, adminPassword, zammadURL, zammadToken string) error {
	schema := tenant.SchemaName("acme")

	// Drop existing tenant if present.
	var exists bool
	_ = db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM global.tenants WHERE slug = 'acme')",
	).Scan(&exists)

	if exists {
		logger.Info("seed-demo: dropping existing acme tenant")
		_, _ = db.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
		_, _ = db.Exec(ctx, "DELETE FROM global.tenants WHERE slug = 'acme'")
	}

	// Re-run the standard seed first.
	if err := Run(ctx, db, databaseURL, migrationsDir, logger, adminPassword, zammadURL, zammadToken); err != nil {
		return fmt.Errorf("running base seed: %w", err)
	}

	// Acquire a tenant-scoped connection for demo data.
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	// Look up SLA policy IDs for linking.
	policyIDs := make(map[string]uuid.UUID)
	rows, err := conn.Query(ctx, "SELECT id, priority FROM sla_policies WHERE is_default = true")
	if err != nil {
		return fmt.Errorf("querying SLA policies: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var priority string
		if err := rows.Scan(&id, &priority); err != nil {
			return fmt.Errorf("scanning SLA policy: %w", err)
		}
		policyIDs[priority] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating SLA policies: %w", err)
	}

	// Seed 10 ticket_meta rows.
	tickets := []struct {
		zammadID     int
		zammadNumber string
		priority     string
	}{
		{1, "10001", "critical"},
		{2, "10002", "high"},
		{3, "10003", "normal"},
		{4, "10004", "low"},
		{5, "10005", "critical"},
		{6, "10006", "high"},
		{7, "10007", "normal"},
		{8, "10008", "normal"},
		{9, "10009", "low"},
		{10, "10010", "high"},
	}

	now := time.Now()
	demoUser := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ticketMetaIDs := make([]uuid.UUID, 0, len(tickets))

	for _, t := range tickets {
		var metaID uuid.UUID
		err := conn.QueryRow(ctx,
			`INSERT INTO ticket_meta (zammad_id, zammad_number, sla_policy_id)
			 VALUES ($1, $2, $3) RETURNING id`,
			t.zammadID, t.zammadNumber, policyIDs[t.priority],
		).Scan(&metaID)
		if err != nil {
			return fmt.Errorf("inserting ticket_meta %d: %w", t.zammadID, err)
		}
		ticketMetaIDs = append(ticketMetaIDs, metaID)
	}

	// Seed SLA states for each ticket.
	slaStates := []struct {
		index int
		state string
	}{
		{0, "breached"},
		{1, "warning"},
		{2, "on_track"},
		{3, "on_track"},
		{4, "breached"},
		{5, "on_track"},
		{6, "met"},
		{7, "on_track"},
		{8, "on_track"},
		{9, "warning"},
	}

	for _, s := range slaStates {
		responseDue := now.Add(time.Duration(30*(s.index+1)) * time.Minute)
		resolutionDue := now.Add(time.Duration(4*(s.index+1)) * time.Hour)

		_, err := conn.Exec(ctx,
			`INSERT INTO sla_states (ticket_meta_id, response_due_at, resolution_due_at, state)
			 VALUES ($1, $2, $3, $4)`,
			ticketMetaIDs[s.index], responseDue, resolutionDue, s.state,
		)
		if err != nil {
			return fmt.Errorf("inserting SLA state for ticket %d: %w", s.index, err)
		}
	}

	// Seed incident links for some tickets.
	incidentLinks := []struct {
		ticketIdx    int
		incidentSlug string
	}{
		{0, "inc-payment-crash-001"},
		{1, "inc-auth-timeout-002"},
		{4, "inc-db-connection-003"},
		{5, "inc-ingress-502-004"},
		{9, "inc-cert-expiry-005"},
	}

	for _, l := range incidentLinks {
		incidentID := uuid.New()
		_, err := conn.Exec(ctx,
			`INSERT INTO incident_links (ticket_meta_id, incident_id, incident_slug, linked_by)
			 VALUES ($1, $2, $3, $4)`,
			ticketMetaIDs[l.ticketIdx], incidentID, l.incidentSlug, demoUser,
		)
		if err != nil {
			return fmt.Errorf("inserting incident link %s: %w", l.incidentSlug, err)
		}
	}

	logger.Info("seed-demo: demo data inserted",
		"tickets", len(tickets),
		"sla_states", len(slaStates),
		"incident_links", len(incidentLinks),
	)

	return nil
}
