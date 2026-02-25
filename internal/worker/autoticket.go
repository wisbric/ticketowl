package worker

import (
	"strings"

	"github.com/wisbric/ticketowl/internal/webhook"
)

// AutoTicketRule mirrors the auto_ticket_rules DB table.
type AutoTicketRule struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	AlertGroup      string `json:"alert_group"`  // exact or prefix match (trailing '-')
	MinSeverity     string `json:"min_severity"` // threshold: low < medium < high < critical
	DefaultPriority string `json:"default_priority"`
	DefaultGroup    string `json:"default_group"`
	TitleTemplate   string `json:"title_template"`
}

// severityRank maps severity strings to numeric ranks for comparison.
var severityRank = map[string]int{
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// MatchRule is a pure function that evaluates whether a NightOwl incident
// matches a single auto-ticket rule. A disabled rule never matches.
func MatchRule(rule AutoTicketRule, incident webhook.NightOwlEvent) bool {
	if !rule.Enabled {
		return false
	}

	// Alert group matching: exact match or prefix match (e.g. "kubernetes-" matches "kubernetes-pod-crash").
	if rule.AlertGroup != "" {
		if strings.HasSuffix(rule.AlertGroup, "-") {
			// Prefix match.
			if !strings.HasPrefix(incident.Service, rule.AlertGroup) {
				return false
			}
		} else {
			// Exact match.
			if incident.Service != rule.AlertGroup {
				return false
			}
		}
	}

	// Severity threshold: incident severity must be >= rule min_severity.
	if rule.MinSeverity != "" {
		incidentRank := severityRank[strings.ToLower(incident.Severity)]
		ruleRank := severityRank[strings.ToLower(rule.MinSeverity)]
		if incidentRank < ruleRank {
			return false
		}
	}

	return true
}

// FindMatchingRules is a pure function that returns all rules that match
// the given incident. It filters disabled rules and evaluates constraints.
func FindMatchingRules(rules []AutoTicketRule, incident webhook.NightOwlEvent) []AutoTicketRule {
	var matched []AutoTicketRule
	for _, r := range rules {
		if MatchRule(r, incident) {
			matched = append(matched, r)
		}
	}
	return matched
}

// RenderTitle renders the title template for an auto-created ticket.
// Supports simple {{.AlertName}} and {{.Summary}} placeholders.
func RenderTitle(template string, incident webhook.NightOwlEvent) string {
	title := template
	title = strings.ReplaceAll(title, "{{.AlertName}}", incident.Slug)
	title = strings.ReplaceAll(title, "{{.Summary}}", incident.Summary)
	title = strings.ReplaceAll(title, "{{.Service}}", incident.Service)
	title = strings.ReplaceAll(title, "{{.Severity}}", incident.Severity)
	return title
}
