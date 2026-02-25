import type { ThreadEntry } from "@/types/api";
import { cn, formatRelativeTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";

interface ThreadViewProps {
  entries: ThreadEntry[];
  showInternal?: boolean;
}

export function ThreadView({ entries, showInternal = true }: ThreadViewProps) {
  const filtered = showInternal ? entries : entries.filter((e) => !e.internal);

  if (filtered.length === 0) {
    return <p className="py-4 text-center text-sm text-muted-foreground">No messages yet.</p>;
  }

  return (
    <div className="space-y-3">
      {filtered.map((entry) => (
        <Card
          key={entry.id}
          className={cn(
            "p-4",
            entry.internal && "border-l-2 border-l-sla-paused bg-muted/30"
          )}
        >
          <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-medium text-foreground">{entry.created_by || entry.sender}</span>
            <span>&middot;</span>
            <span>{formatRelativeTime(entry.created_at)}</span>
            {entry.internal && (
              <>
                <span>&middot;</span>
                <span className="text-sla-paused">Internal</span>
              </>
            )}
            {entry.type !== "note" && entry.type !== "internal_note" && (
              <>
                <span>&middot;</span>
                <span className="capitalize">{entry.type}</span>
              </>
            )}
          </div>
          <div
            className="prose prose-sm max-w-none text-foreground prose-p:my-1"
            dangerouslySetInnerHTML={{ __html: entry.body }}
          />
        </Card>
      ))}
    </div>
  );
}
