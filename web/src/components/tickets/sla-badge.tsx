import { Badge } from "@/components/ui/badge";
import { cn, slaStateDot, slaStateLabel } from "@/lib/utils";

interface SLABadgeProps {
  state: string;
  className?: string;
}

export function SLABadge({ state, className }: SLABadgeProps) {
  return (
    <Badge variant="secondary" className={cn("gap-1.5", className)}>
      <span className={cn("inline-block h-2 w-2 rounded-full", slaStateDot(state))} />
      {slaStateLabel(state)}
    </Badge>
  );
}
