import { useState, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { cn, slaStateColor, slaStateLabel } from "@/lib/utils";
import type { SLAState } from "@/types/api";

interface SLATimerProps {
  sla: SLAState;
}

function formatCountdown(targetDate: string): string {
  const now = new Date().getTime();
  const target = new Date(targetDate).getTime();
  const diff = target - now;

  if (diff <= 0) return "Overdue";

  const hours = Math.floor(diff / (1000 * 60 * 60));
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
  const seconds = Math.floor((diff % (1000 * 60)) / 1000);

  if (hours > 24) {
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h`;
  }
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m ${seconds}s`;
}

export function SLATimer({ sla }: SLATimerProps) {
  const [, setTick] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => setTick((t) => t + 1), 1000);
    return () => clearInterval(interval);
  }, []);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">SLA Status</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">State</span>
          <span className={cn("text-sm font-medium", slaStateColor(sla.state))}>
            {slaStateLabel(sla.state)}
            {sla.paused && " (Paused)"}
          </span>
        </div>

        {sla.response_due_at && !sla.response_met_at && (
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">Response due</span>
            <span className="text-sm font-mono font-medium">
              {formatCountdown(sla.response_due_at)}
            </span>
          </div>
        )}

        {sla.response_met_at && (
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">Response</span>
            <span className="text-sm text-sla-on-track">Met</span>
          </div>
        )}

        {sla.resolution_due_at && (
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">Resolution due</span>
            <span className="text-sm font-mono font-medium">
              {formatCountdown(sla.resolution_due_at)}
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
