package worker_test

import (
	"testing"

	"github.com/wisbric/ticketowl/internal/webhook"
	"github.com/wisbric/ticketowl/internal/worker"
)

func TestMatchRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     worker.AutoTicketRule
		incident webhook.NightOwlEvent
		want     bool
	}{
		{
			name: "exact match on alert_group",
			rule: worker.AutoTicketRule{
				Enabled:    true,
				AlertGroup: "api-gateway",
			},
			incident: webhook.NightOwlEvent{
				Event:   "incident.created",
				Service: "api-gateway",
			},
			want: true,
		},
		{
			name: "exact match fails on different service",
			rule: worker.AutoTicketRule{
				Enabled:    true,
				AlertGroup: "api-gateway",
			},
			incident: webhook.NightOwlEvent{
				Event:   "incident.created",
				Service: "database",
			},
			want: false,
		},
		{
			name: "prefix match on alert_group",
			rule: worker.AutoTicketRule{
				Enabled:    true,
				AlertGroup: "kubernetes-",
			},
			incident: webhook.NightOwlEvent{
				Event:   "incident.created",
				Service: "kubernetes-pod-crash",
			},
			want: true,
		},
		{
			name: "prefix match fails on non-matching prefix",
			rule: worker.AutoTicketRule{
				Enabled:    true,
				AlertGroup: "kubernetes-",
			},
			incident: webhook.NightOwlEvent{
				Event:   "incident.created",
				Service: "api-gateway",
			},
			want: false,
		},
		{
			name: "severity threshold — incident meets minimum",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				MinSeverity: "high",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Severity: "critical",
			},
			want: true,
		},
		{
			name: "severity threshold — incident equals minimum",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				MinSeverity: "high",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Severity: "high",
			},
			want: true,
		},
		{
			name: "severity threshold — incident below minimum",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				MinSeverity: "high",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Severity: "medium",
			},
			want: false,
		},
		{
			name: "disabled rule never matches",
			rule: worker.AutoTicketRule{
				Enabled:    false,
				AlertGroup: "api-gateway",
			},
			incident: webhook.NightOwlEvent{
				Event:   "incident.created",
				Service: "api-gateway",
			},
			want: false,
		},
		{
			name: "combined constraints — alert_group + severity, both match",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				AlertGroup:  "kubernetes-",
				MinSeverity: "high",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Service:  "kubernetes-pod-crash",
				Severity: "critical",
			},
			want: true,
		},
		{
			name: "combined constraints — alert_group matches, severity fails",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				AlertGroup:  "kubernetes-",
				MinSeverity: "high",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Service:  "kubernetes-pod-crash",
				Severity: "low",
			},
			want: false,
		},
		{
			name: "combined constraints — severity matches, alert_group fails",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				AlertGroup:  "api-gateway",
				MinSeverity: "medium",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Service:  "database",
				Severity: "critical",
			},
			want: false,
		},
		{
			name: "no constraints (enabled, empty alert_group, empty severity) — matches everything",
			rule: worker.AutoTicketRule{
				Enabled: true,
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Service:  "anything",
				Severity: "low",
			},
			want: true,
		},
		{
			name: "unknown severity in incident — treated as rank 0",
			rule: worker.AutoTicketRule{
				Enabled:     true,
				MinSeverity: "low",
			},
			incident: webhook.NightOwlEvent{
				Event:    "incident.created",
				Severity: "unknown",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := worker.MatchRule(tt.rule, tt.incident)
			if got != tt.want {
				t.Errorf("MatchRule = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindMatchingRules(t *testing.T) {
	rules := []worker.AutoTicketRule{
		{
			Name:       "Kubernetes alerts",
			Enabled:    true,
			AlertGroup: "kubernetes-",
		},
		{
			Name:        "Critical only",
			Enabled:     true,
			MinSeverity: "critical",
		},
		{
			Name:    "Disabled rule",
			Enabled: false,
		},
		{
			Name:        "API gateway high+",
			Enabled:     true,
			AlertGroup:  "api-gateway",
			MinSeverity: "high",
		},
	}

	tests := []struct {
		name     string
		incident webhook.NightOwlEvent
		want     int // number of matching rules
	}{
		{
			name: "k8s critical matches two rules",
			incident: webhook.NightOwlEvent{
				Service:  "kubernetes-pod-crash",
				Severity: "critical",
			},
			want: 2, // "Kubernetes alerts" + "Critical only"
		},
		{
			name: "api-gateway critical matches two rules",
			incident: webhook.NightOwlEvent{
				Service:  "api-gateway",
				Severity: "critical",
			},
			want: 2, // "Critical only" + "API gateway high+"
		},
		{
			name: "api-gateway medium matches zero rules",
			incident: webhook.NightOwlEvent{
				Service:  "api-gateway",
				Severity: "medium",
			},
			want: 0,
		},
		{
			name: "database low matches zero rules",
			incident: webhook.NightOwlEvent{
				Service:  "database",
				Severity: "low",
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := worker.FindMatchingRules(rules, tt.incident)
			if len(matched) != tt.want {
				names := make([]string, len(matched))
				for i, m := range matched {
					names[i] = m.Name
				}
				t.Errorf("FindMatchingRules matched %d rules %v, want %d", len(matched), names, tt.want)
			}
		})
	}
}

func TestRenderTitle(t *testing.T) {
	tests := []struct {
		name     string
		template string
		incident webhook.NightOwlEvent
		want     string
	}{
		{
			name:     "default template",
			template: "{{.AlertName}}: {{.Summary}}",
			incident: webhook.NightOwlEvent{
				Slug:    "INC-001",
				Summary: "Server down",
			},
			want: "INC-001: Server down",
		},
		{
			name:     "service and severity",
			template: "[{{.Severity}}] {{.Service}} — {{.Summary}}",
			incident: webhook.NightOwlEvent{
				Severity: "critical",
				Service:  "api-gateway",
				Summary:  "High error rate",
			},
			want: "[critical] api-gateway — High error rate",
		},
		{
			name:     "no placeholders",
			template: "Auto-created ticket",
			incident: webhook.NightOwlEvent{},
			want:     "Auto-created ticket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := worker.RenderTitle(tt.template, tt.incident)
			if got != tt.want {
				t.Errorf("RenderTitle = %q, want %q", got, tt.want)
			}
		})
	}
}
