import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatRelativeTime(date: string | Date): string {
  const now = new Date();
  const d = typeof date === "string" ? new Date(date) : date;
  const diff = now.getTime() - d.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return d.toLocaleDateString();
}

export function slaStateColor(state: string): string {
  switch (state.toLowerCase()) {
    case "on_track":
      return "text-sla-on-track";
    case "warning":
      return "text-sla-warning";
    case "breached":
      return "text-sla-breached";
    case "met":
      return "text-sla-met";
    case "paused":
      return "text-sla-paused";
    default:
      return "text-muted-foreground";
  }
}

export function slaStateDot(state: string): string {
  switch (state.toLowerCase()) {
    case "on_track":
      return "bg-sla-on-track";
    case "warning":
      return "bg-sla-warning";
    case "breached":
      return "bg-sla-breached";
    case "met":
      return "bg-sla-met";
    case "paused":
      return "bg-sla-paused";
    default:
      return "bg-muted-foreground";
  }
}

export function priorityColor(priority: string): string {
  switch (priority.toLowerCase()) {
    case "urgent":
    case "1 urgent":
      return "text-priority-urgent";
    case "high":
    case "2 high":
      return "text-priority-high";
    case "normal":
    case "3 normal":
      return "text-priority-normal";
    case "low":
    case "4 low":
      return "text-priority-low";
    default:
      return "text-muted-foreground";
  }
}

export function slaStateLabel(state: string): string {
  switch (state.toLowerCase()) {
    case "on_track":
      return "On Track";
    case "warning":
      return "Warning";
    case "breached":
      return "Breached";
    case "met":
      return "Met";
    case "paused":
      return "Paused";
    default:
      return state;
  }
}
