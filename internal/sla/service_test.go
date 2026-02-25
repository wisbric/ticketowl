package sla_test

import (
	"testing"
	"time"

	"github.com/wisbric/ticketowl/internal/sla"
)

func TestComputeState(t *testing.T) {
	// Base time for all tests.
	base := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		input        sla.ComputeInput
		wantLabel    sla.StateLabel
		wantPaused   bool
		wantRespGT0  bool // response seconds remaining > 0
		wantResolGT0 bool // resolution seconds remaining > 0
	}{
		{
			name: "10 min elapsed, 60 min response SLA → on_track",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480,
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				Now:               base.Add(10 * time.Minute),
			},
			wantLabel:    sla.LabelOnTrack,
			wantRespGT0:  true,
			wantResolGT0: true,
		},
		{
			name: "50 min elapsed, 60 min response SLA (< 20% remaining) → warning",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480,
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				Now:               base.Add(50 * time.Minute),
			},
			wantLabel:    sla.LabelWarning,
			wantRespGT0:  true,
			wantResolGT0: true,
		},
		{
			name: "75 min elapsed, 60 min response SLA → breached",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480,
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				Now:               base.Add(75 * time.Minute),
			},
			wantLabel:    sla.LabelBreached,
			wantRespGT0:  false, // response remaining = 0
			wantResolGT0: true,
		},
		{
			name: "9h elapsed, 8h resolution SLA, response met → breached",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480, // 8 hours
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				ResponseMetAt:     timePtr(base.Add(30 * time.Minute)), // response met at 30 min
				Now:               base.Add(9 * time.Hour),
			},
			wantLabel:    sla.LabelBreached,
			wantRespGT0:  false,
			wantResolGT0: false,
		},
		{
			name: "70 min elapsed, 30 min paused, 60 min response SLA → on_track (effective 40 min)",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480,
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				AccumulatedPause:  30 * 60, // 30 minutes in seconds
				Now:               base.Add(70 * time.Minute),
			},
			wantLabel:    sla.LabelOnTrack,
			wantRespGT0:  true,
			wantResolGT0: true,
		},
		{
			name: "Both response and resolution met before deadlines → met",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480,
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				ResponseMetAt:     timePtr(base.Add(20 * time.Minute)),
				Now:               base.Add(2 * time.Hour), // well within 8h resolution
			},
			wantLabel:    sla.LabelMet,
			wantRespGT0:  false, // response was already met
			wantResolGT0: true,
		},
		{
			name: "Ticket paused → paused flag true",
			input: sla.ComputeInput{
				ResponseMinutes:   60,
				ResolutionMinutes: 480,
				WarningThreshold:  0.20,
				TicketCreatedAt:   base,
				Paused:            true,
				Now:               base.Add(10 * time.Minute),
			},
			wantLabel:    sla.LabelOnTrack,
			wantPaused:   true,
			wantRespGT0:  true,
			wantResolGT0: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sla.ComputeState(tt.input)

			if result.Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", result.Label, tt.wantLabel)
			}

			if result.Paused != tt.wantPaused {
				t.Errorf("Paused = %v, want %v", result.Paused, tt.wantPaused)
			}

			if tt.wantRespGT0 && result.ResponseSecondsRemaining <= 0 {
				t.Errorf("ResponseSecondsRemaining = %d, want > 0", result.ResponseSecondsRemaining)
			}
			if !tt.wantRespGT0 && result.ResponseSecondsRemaining > 0 {
				t.Errorf("ResponseSecondsRemaining = %d, want 0", result.ResponseSecondsRemaining)
			}

			if tt.wantResolGT0 && result.ResolutionSecondsRemaining <= 0 {
				t.Errorf("ResolutionSecondsRemaining = %d, want > 0", result.ResolutionSecondsRemaining)
			}
			if !tt.wantResolGT0 && result.ResolutionSecondsRemaining > 0 {
				t.Errorf("ResolutionSecondsRemaining = %d, want 0", result.ResolutionSecondsRemaining)
			}
		})
	}
}

func TestComputeState_ResponseDueAt(t *testing.T) {
	base := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	result := sla.ComputeState(sla.ComputeInput{
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
		TicketCreatedAt:   base,
		Now:               base.Add(10 * time.Minute),
	})

	expectedResp := base.Add(60 * time.Minute)
	expectedResol := base.Add(480 * time.Minute)

	if !result.ResponseDueAt.Equal(expectedResp) {
		t.Errorf("ResponseDueAt = %v, want %v", result.ResponseDueAt, expectedResp)
	}
	if !result.ResolutionDueAt.Equal(expectedResol) {
		t.Errorf("ResolutionDueAt = %v, want %v", result.ResolutionDueAt, expectedResol)
	}
}

func TestComputeState_WarningThresholdEdge(t *testing.T) {
	base := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	// Exactly at the warning threshold boundary: 48 min elapsed, 60 min SLA, 20% threshold.
	// Warning threshold = 0.20 * 3600 = 720 seconds = 12 minutes remaining.
	// At 48 min elapsed: remaining = 12 min = 720 sec. This is exactly the threshold.
	// remaining < threshold means < 720, so at exactly 720 it should still be on_track.
	result := sla.ComputeState(sla.ComputeInput{
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
		TicketCreatedAt:   base,
		Now:               base.Add(48 * time.Minute),
	})

	if result.Label != sla.LabelOnTrack {
		t.Errorf("Label = %q, want %q (exactly at threshold boundary)", result.Label, sla.LabelOnTrack)
	}

	// 49 min elapsed: remaining = 11 min = 660 sec < 720, should be warning.
	result2 := sla.ComputeState(sla.ComputeInput{
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
		TicketCreatedAt:   base,
		Now:               base.Add(49 * time.Minute),
	})

	if result2.Label != sla.LabelWarning {
		t.Errorf("Label = %q, want %q (just past threshold)", result2.Label, sla.LabelWarning)
	}
}

func TestComputeState_PauseReducesEffectiveElapsed(t *testing.T) {
	base := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	// Without pause: 65 min elapsed, 60 min SLA → breached.
	breached := sla.ComputeState(sla.ComputeInput{
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
		TicketCreatedAt:   base,
		Now:               base.Add(65 * time.Minute),
	})
	if breached.Label != sla.LabelBreached {
		t.Fatalf("expected breached without pause, got %q", breached.Label)
	}

	// With 10 min pause: effective = 55 min → warning (< 20% of 60 min = 12 min remaining).
	withPause := sla.ComputeState(sla.ComputeInput{
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
		TicketCreatedAt:   base,
		AccumulatedPause:  10 * 60,
		Now:               base.Add(65 * time.Minute),
	})
	if withPause.Label != sla.LabelWarning {
		t.Errorf("Label = %q, want %q (10 min pause should reduce to warning)", withPause.Label, sla.LabelWarning)
	}
}

func TestCheckBreach(t *testing.T) {
	tests := []struct {
		name  string
		state sla.State
		want  bool
	}{
		{
			name:  "breached with no prior alert → true",
			state: sla.State{Label: sla.LabelBreached, FirstBreachAlertedAt: nil},
			want:  true,
		},
		{
			name:  "breached with prior alert → false",
			state: sla.State{Label: sla.LabelBreached, FirstBreachAlertedAt: timePtr(time.Now())},
			want:  false,
		},
		{
			name:  "on_track → false",
			state: sla.State{Label: sla.LabelOnTrack},
			want:  false,
		},
		{
			name:  "warning → false",
			state: sla.State{Label: sla.LabelWarning},
			want:  false,
		},
		{
			name:  "met → false",
			state: sla.State{Label: sla.LabelMet},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sla.CheckBreach(&tt.state)
			if got != tt.want {
				t.Errorf("CheckBreach = %v, want %v", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
