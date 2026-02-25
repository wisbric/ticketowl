# 08 — SLA

## Overview

SLA policies define how quickly tickets must receive a first response and reach resolution, based on their priority. TicketOwl tracks the state of every open ticket against its policy and escalates through NightOwl when SLA is breached.

---

## SLA Policies

Each policy is scoped to a priority level. A tenant can have multiple policies (e.g., one per customer tier), but each priority has at most one `is_default` policy.

```
Priority | Response | Resolution
---------|----------|------------
critical |   30 min |    4 hours
high     |   60 min |    8 hours
normal   |    4 hrs |   24 hours
low      |    8 hrs |   48 hours
```

These are suggested defaults. Tenant admins set their own values.

---

## SLA State Machine

```
         ┌──────────┐
   ───►  │ on_track │
         └────┬─────┘
              │ < warning_threshold remaining
              ▼
         ┌─────────┐
         │ warning │
         └────┬────┘
              │ deadline passed
              ▼
         ┌─────────┐
         │ breached│ ──► NightOwl alert fired once
         └─────────┘
              
         ┌─────┐
         │ met │  ◄── response and resolution both completed before deadline
         └─────┘

         ┌────────┐
         │ paused │  ◄── ticket in a pause_status (e.g. "pending customer")
         └────────┘       SLA clock stops accumulating
```

State transitions:
- `on_track` → `warning` when remaining time < `warning_threshold * total_time`
- `warning` → `breached` when current time > due_at
- Any → `met` when both response and resolution are recorded before their deadlines
- Any → `paused` when ticket enters a `pause_status`; resumes from same state when it leaves

---

## SLA Computation

`sla.ComputeState()` is a **pure function** with no side effects:

```
Input:
  - policy (response_minutes, resolution_minutes, warning_threshold)
  - ticket_created_at
  - response_met_at (nullable)
  - accumulated_pause_secs
  - paused (bool)
  - now

Output:
  - label (on_track | warning | breached | met)
  - response_due_at
  - resolution_due_at
  - response_seconds_remaining
  - resolution_seconds_remaining
  - paused (bool)
```

Effective elapsed time = (now - created_at) - accumulated_pause_secs.
Deadlines are computed from created_at + policy minutes (pause does not shift the clock forward — it reduces effective elapsed time).

---

## Worker — SLA Poller

Runs every 60 seconds:

1. Query `sla_states` for tickets in `on_track` or `warning` where `updated_at < now - 55s`
2. For each: fetch current ticket from Zammad (to get current status and timestamps)
3. Check if ticket status is in `zammad_config.pause_statuses`
   - If newly paused: record `paused_at`, set `paused = true`
   - If newly resumed: add `(now - paused_at)` to `accumulated_pause_secs`, set `paused = false`
4. Call `sla.ComputeState()` with current inputs
5. Update `sla_states` row
6. If state is `breached` and `first_breach_alerted_at IS NULL`:
   - Call `notification.Service.AlertSLABreach()`
   - Set `first_breach_alerted_at = now`

---

## Test Cases for SLA Computation

These cases must be covered in `internal/sla/service_test.go` (table-driven):

| Scenario | Expected state |
|----------|----------------|
| 10 min elapsed, 60 min response SLA | on_track |
| 50 min elapsed, 60 min response SLA (< 20% remaining) | warning |
| 75 min elapsed, 60 min response SLA | breached |
| 9h elapsed, 8h resolution SLA, response met | breached |
| 70 min elapsed, but 30 min paused, 60 min response SLA | on_track |
| Both response and resolution met before deadlines | met |
| Ticket paused | paused flag = true |
