package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/ticketowl/internal/admin"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/sla"
	"github.com/wisbric/ticketowl/internal/ticket"
	"github.com/wisbric/ticketowl/internal/webhook"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// derefString safely dereferences a *string, returning "" if nil.
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// DefaultEventHandler implements EventHandler by dispatching to the appropriate
// business logic for each event type.
type DefaultEventHandler struct {
	pool   *pgxpool.Pool
	schema string
	logger *slog.Logger
}

// NewDefaultEventHandler creates a default event handler.
func NewDefaultEventHandler(pool *pgxpool.Pool, schema string, logger *slog.Logger) *DefaultEventHandler {
	return &DefaultEventHandler{pool: pool, schema: schema, logger: logger}
}

// acquireConn acquires a connection from the pool and sets the tenant search_path.
func (h *DefaultEventHandler) acquireConn(ctx context.Context) (*pgxpool.Conn, error) {
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", h.schema)); err != nil {
		conn.Release()
		return nil, fmt.Errorf("setting search_path: %w", err)
	}
	return conn, nil
}

// zammadClient creates a Zammad client from the tenant's config.
func (h *DefaultEventHandler) zammadClient(ctx context.Context, conn *pgxpool.Conn) (*zammad.Client, error) {
	adminStore := admin.NewStore(conn)
	cfg, err := adminStore.GetZammadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting zammad config: %w", err)
	}
	return zammad.New(cfg.URL, cfg.APIToken, zammad.WithLogger(h.logger)), nil
}

// nightowlClient creates a NightOwl client from the tenant's integration key config.
func (h *DefaultEventHandler) nightowlClient(ctx context.Context, conn *pgxpool.Conn) (*nightowl.Client, error) {
	adminStore := admin.NewStore(conn)
	key, err := adminStore.GetIntegrationKey(ctx, "nightowl")
	if err != nil {
		return nil, fmt.Errorf("getting nightowl integration key: %w", err)
	}
	return nightowl.New(key.APIURL, key.APIKey, nightowl.WithLogger(h.logger)), nil
}

// autoAssignTicket looks up the group-roster mapping for the given Zammad group,
// fetches the current on-call from NightOwl, and assigns the ticket in Zammad.
func (h *DefaultEventHandler) autoAssignTicket(ctx context.Context, conn *pgxpool.Conn, zClient *zammad.Client, zammadTicketID int, zammadGroup string) {
	if zammadGroup == "" {
		return
	}

	adminStore := admin.NewStore(conn)
	mapping, err := adminStore.GetGroupRosterMappingByZammadGroup(ctx, zammadGroup)
	if err != nil {
		if err == pgx.ErrNoRows {
			return // No mapping for this group
		}
		h.logger.Warn("looking up group-roster mapping", "error", err, "group", zammadGroup)
		return
	}

	if !mapping.AutoAssign {
		return
	}

	// Fetch on-call from NightOwl.
	noClient, err := h.nightowlClient(ctx, conn)
	if err != nil {
		h.logger.Warn("creating nightowl client for auto-assign", "error", err)
		return
	}

	oncall, err := noClient.GetOnCallForService(ctx, mapping.RosterID)
	if err != nil {
		h.logger.Warn("fetching on-call for auto-assign",
			"error", err,
			"roster_id", mapping.RosterID,
			"group", zammadGroup,
		)
		return
	}

	// Look up the on-call person's Zammad user by email.
	zUser, err := zClient.SearchUsersByEmail(ctx, oncall.UserEmail)
	if err != nil {
		h.logger.Warn("searching zammad user by email for auto-assign",
			"error", err,
			"email", oncall.UserEmail,
		)
		return
	}
	if zUser == nil {
		h.logger.Info("on-call user not found in Zammad, skipping auto-assign",
			"email", oncall.UserEmail,
			"group", zammadGroup,
		)
		return
	}

	// Update ticket owner in Zammad.
	if _, err := zClient.UpdateTicket(ctx, zammadTicketID, zammad.TicketUpdateRequest{
		OwnerID: &zUser.ID,
	}); err != nil {
		h.logger.Error("auto-assigning ticket to on-call",
			"error", err,
			"zammad_ticket_id", zammadTicketID,
			"owner_id", zUser.ID,
		)
		return
	}

	h.logger.Info("auto-assigned ticket to on-call",
		"zammad_ticket_id", zammadTicketID,
		"owner", oncall.UserName,
		"email", oncall.UserEmail,
		"group", zammadGroup,
	)
}

// HandleZammadEvent processes a Zammad webhook event.
func (h *DefaultEventHandler) HandleZammadEvent(ctx context.Context, evt webhook.ZammadEvent) error {
	switch evt.Event {
	case "ticket.create":
		return h.handleTicketCreate(ctx, evt)
	case "ticket.update":
		return h.handleTicketUpdate(ctx, evt)
	case "article.create":
		h.logger.Info("processing article.create event", "ticket_id", evt.TicketID)
		return nil
	default:
		return fmt.Errorf("unknown zammad event type: %s", evt.Event)
	}
}

// HandleNightOwlEvent processes a NightOwl webhook event.
func (h *DefaultEventHandler) HandleNightOwlEvent(ctx context.Context, evt webhook.NightOwlEvent) error {
	switch evt.Event {
	case "incident.created":
		return h.handleIncidentCreated(ctx, evt)
	case "incident.resolved":
		return h.handleIncidentResolved(ctx, evt)
	default:
		return fmt.Errorf("unknown nightowl event type: %s", evt.Event)
	}
}

// handleTicketCreate upserts ticket_meta and initializes SLA tracking.
func (h *DefaultEventHandler) handleTicketCreate(ctx context.Context, evt webhook.ZammadEvent) error {
	h.logger.Info("processing ticket.create", "ticket_id", evt.TicketID, "number", evt.Number)

	conn, err := h.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// Upsert ticket_meta.
	ticketStore := ticket.NewStore(conn)
	meta, err := ticketStore.Upsert(ctx, evt.TicketID, evt.Number, nil)
	if err != nil {
		return fmt.Errorf("upserting ticket_meta: %w", err)
	}

	// Find default SLA policy by priority.
	slaStore := sla.NewStore(conn)
	priority := evt.Priority
	if priority == "" {
		priority = "normal"
	}
	policy, err := slaStore.GetDefaultPolicyByPriority(ctx, priority)
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Info("no default SLA policy for priority, skipping SLA init", "priority", priority)
			return nil
		}
		return fmt.Errorf("getting default SLA policy: %w", err)
	}

	// Assign SLA policy to ticket_meta.
	_, err = ticketStore.Upsert(ctx, evt.TicketID, evt.Number, &policy.ID)
	if err != nil {
		return fmt.Errorf("assigning SLA policy to ticket: %w", err)
	}

	// Initialize SLA state with due dates.
	now := time.Now()
	responseDue := now.Add(time.Duration(policy.ResponseMinutes) * time.Minute)
	resolutionDue := now.Add(time.Duration(policy.ResolutionMinutes) * time.Minute)
	state := &sla.State{
		ID:              uuid.New(),
		TicketMetaID:    meta.ID,
		ResponseDueAt:   &responseDue,
		ResolutionDueAt: &resolutionDue,
		Label:           sla.LabelOnTrack,
		UpdatedAt:       now,
	}
	if err := slaStore.UpsertState(ctx, state); err != nil {
		return fmt.Errorf("initializing SLA state: %w", err)
	}

	h.logger.Info("ticket meta created with SLA",
		"ticket_id", evt.TicketID,
		"meta_id", meta.ID,
		"policy", policy.Name,
		"response_due", responseDue,
		"resolution_due", resolutionDue,
	)

	// Auto-assign ticket to on-call if a group-roster mapping exists.
	if evt.Group != "" {
		zClient, err := h.zammadClient(ctx, conn)
		if err != nil {
			h.logger.Warn("creating zammad client for auto-assign", "error", err)
		} else {
			h.autoAssignTicket(ctx, conn, zClient, evt.TicketID, evt.Group)
		}
	}

	return nil
}

// handleTicketUpdate checks for pause status changes and recomputes SLA.
func (h *DefaultEventHandler) handleTicketUpdate(ctx context.Context, evt webhook.ZammadEvent) error {
	h.logger.Info("processing ticket.update", "ticket_id", evt.TicketID, "state", evt.State)

	conn, err := h.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// Look up ticket_meta.
	ticketStore := ticket.NewStore(conn)
	meta, err := ticketStore.GetByZammadID(ctx, evt.TicketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Info("ticket_meta not found for update, skipping", "ticket_id", evt.TicketID)
			return nil
		}
		return fmt.Errorf("getting ticket_meta: %w", err)
	}

	// Look up SLA state.
	slaStore := sla.NewStore(conn)
	state, err := slaStore.GetStateByTicketMetaID(ctx, meta.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // No SLA tracking for this ticket
		}
		return fmt.Errorf("getting SLA state: %w", err)
	}

	// Check if the Zammad state indicates a pause (e.g., "pending reminder", "pending close").
	isPauseState := evt.State == "pending reminder" || evt.State == "pending close"
	now := time.Now()

	if isPauseState && !state.Paused {
		// Pause the SLA clock.
		state.Paused = true
		state.PausedAt = &now
		state.UpdatedAt = now
		if err := slaStore.UpsertState(ctx, state); err != nil {
			return fmt.Errorf("pausing SLA state: %w", err)
		}
		h.logger.Info("SLA paused", "ticket_id", evt.TicketID, "state", evt.State)
	} else if !isPauseState && state.Paused {
		// Resume the SLA clock — add paused duration to accumulated.
		if state.PausedAt != nil {
			pausedSecs := int(now.Sub(*state.PausedAt).Seconds())
			state.AccumulatedPauseSecs += pausedSecs

			// Extend due dates by pause duration.
			pauseDuration := time.Duration(pausedSecs) * time.Second
			if state.ResponseDueAt != nil {
				t := state.ResponseDueAt.Add(pauseDuration)
				state.ResponseDueAt = &t
			}
			if state.ResolutionDueAt != nil {
				t := state.ResolutionDueAt.Add(pauseDuration)
				state.ResolutionDueAt = &t
			}
		}
		state.Paused = false
		state.PausedAt = nil
		state.UpdatedAt = now
		if err := slaStore.UpsertState(ctx, state); err != nil {
			return fmt.Errorf("resuming SLA state: %w", err)
		}
		h.logger.Info("SLA resumed", "ticket_id", evt.TicketID, "state", evt.State)
	}

	// If this is a response (first article from agent), mark response as met.
	if state.ResponseMetAt == nil && (evt.State == "open" || evt.State == "new") {
		// Response is tracked separately via article.create in future.
		// For now, we only handle pause/resume here.
	}

	return nil
}

// handleIncidentCreated evaluates auto-ticket rules and creates Zammad tickets.
func (h *DefaultEventHandler) handleIncidentCreated(ctx context.Context, evt webhook.NightOwlEvent) error {
	h.logger.Info("processing incident.created",
		"incident_id", evt.IncidentID,
		"slug", evt.Slug,
		"severity", evt.Severity,
		"service", evt.Service,
	)

	conn, err := h.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// Load auto_ticket_rules.
	adminStore := admin.NewStore(conn)
	dbRules, err := adminStore.ListAutoTicketRules(ctx)
	if err != nil {
		return fmt.Errorf("listing auto_ticket_rules: %w", err)
	}

	// Convert admin.AutoTicketRule to worker.AutoTicketRule for matching.
	rules := make([]AutoTicketRule, len(dbRules))
	for i, r := range dbRules {
		rules[i] = AutoTicketRule{
			ID:              r.ID.String(),
			Name:            r.Name,
			Enabled:         r.Enabled,
			AlertGroup:      derefString(r.AlertGroup),
			MinSeverity:     derefString(r.MinSeverity),
			DefaultPriority: r.DefaultPriority,
			DefaultGroup:    derefString(r.DefaultGroup),
			TitleTemplate:   r.TitleTemplate,
		}
	}

	matched := FindMatchingRules(rules, evt)
	if len(matched) == 0 {
		h.logger.Info("no auto-ticket rules matched", "incident_id", evt.IncidentID)
		return nil
	}

	// Create Zammad client from tenant config.
	zClient, err := h.zammadClient(ctx, conn)
	if err != nil {
		return fmt.Errorf("creating zammad client: %w", err)
	}

	incidentUUID, err := uuid.Parse(evt.IncidentID)
	if err != nil {
		return fmt.Errorf("parsing incident ID: %w", err)
	}

	ticketStore := ticket.NewStore(conn)
	linkStore := link.NewStore(conn)

	for _, rule := range matched {
		title := RenderTitle(rule.TitleTemplate, evt)

		zTicket, err := zClient.CreateTicket(ctx, zammad.TicketCreateRequest{
			Title:      title,
			GroupID:    1, // Default group; real group resolution from rule.DefaultGroup is future work
			CustomerID: 1, // System customer
		})
		if err != nil {
			h.logger.Error("creating zammad ticket from auto-rule",
				"error", err,
				"rule", rule.Name,
				"incident_id", evt.IncidentID,
			)
			continue
		}

		h.logger.Info("auto-ticket created",
			"rule", rule.Name,
			"zammad_ticket_id", zTicket.ID,
			"zammad_number", zTicket.Number,
			"incident_id", evt.IncidentID,
		)

		// Upsert ticket_meta.
		meta, err := ticketStore.Upsert(ctx, zTicket.ID, zTicket.Number, nil)
		if err != nil {
			h.logger.Error("upserting ticket_meta for auto-ticket", "error", err, "zammad_id", zTicket.ID)
			continue
		}

		// Link incident to ticket.
		if _, err := linkStore.CreateIncidentLink(ctx, meta.ID, incidentUUID, uuid.Nil, evt.Slug); err != nil {
			h.logger.Error("creating incident link", "error", err, "incident_id", evt.IncidentID)
		}

		// Auto-assign ticket to on-call if a group-roster mapping exists.
		if zTicket.Group != "" {
			h.autoAssignTicket(ctx, conn, zClient, zTicket.ID, zTicket.Group)
		} else if rule.DefaultGroup != "" {
			h.autoAssignTicket(ctx, conn, zClient, zTicket.ID, rule.DefaultGroup)
		}
	}

	return nil
}

// handleIncidentResolved finds linked tickets and closes them in Zammad.
func (h *DefaultEventHandler) handleIncidentResolved(ctx context.Context, evt webhook.NightOwlEvent) error {
	h.logger.Info("processing incident.resolved",
		"incident_id", evt.IncidentID,
		"slug", evt.Slug,
	)

	conn, err := h.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	incidentUUID, err := uuid.Parse(evt.IncidentID)
	if err != nil {
		return fmt.Errorf("parsing incident ID: %w", err)
	}

	// Find tickets linked to this incident.
	linkStore := link.NewStore(conn)
	links, err := linkStore.ListIncidentLinksByIncidentID(ctx, incidentUUID)
	if err != nil {
		return fmt.Errorf("listing incident links: %w", err)
	}

	if len(links) == 0 {
		h.logger.Info("no linked tickets for resolved incident", "incident_id", evt.IncidentID)
		return nil
	}

	// Create Zammad client from tenant config.
	zClient, err := h.zammadClient(ctx, conn)
	if err != nil {
		return fmt.Errorf("creating zammad client: %w", err)
	}

	// Close each linked ticket in Zammad (state_id=4 is "closed" in default Zammad).
	closedStateID := 4
	for _, lnk := range links {
		// Get Zammad ticket ID from ticket_meta.
		ticketStore := ticket.NewStore(conn)
		meta, err := ticketStore.GetByID(ctx, lnk.TicketMetaID)
		if err != nil {
			h.logger.Error("getting ticket_meta for incident close", "error", err, "meta_id", lnk.TicketMetaID)
			continue
		}

		if _, err := zClient.UpdateTicket(ctx, meta.ZammadID, zammad.TicketUpdateRequest{
			StateID: &closedStateID,
		}); err != nil {
			h.logger.Error("closing zammad ticket",
				"error", err,
				"zammad_id", meta.ZammadID,
				"incident_id", evt.IncidentID,
			)
			continue
		}

		h.logger.Info("closed zammad ticket for resolved incident",
			"zammad_id", meta.ZammadID,
			"zammad_number", meta.ZammadNumber,
			"incident_id", evt.IncidentID,
		)
	}

	return nil
}
