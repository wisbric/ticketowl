import { useQuery } from "@tanstack/react-query";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { api } from "@/lib/api";
import type { Suggestion } from "@/types/api";
import { BookOpen } from "lucide-react";

interface SuggestionsPanelProps {
  ticketId: number;
}

export function SuggestionsPanel({ ticketId }: SuggestionsPanelProps) {
  const { data: suggestions } = useQuery({
    queryKey: ["ticket-suggestions", ticketId],
    queryFn: () => api.get<Suggestion[]>(`/tickets/${ticketId}/suggestions`),
  });

  if (!suggestions || suggestions.length === 0) return null;

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-sm">
          <BookOpen className="h-4 w-4" />
          Suggested Articles
        </CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="space-y-2">
          {suggestions.map((s) => (
            <li key={s.id}>
              <a
                href={s.url}
                target="_blank"
                rel="noopener noreferrer"
                className="block rounded-md p-2 text-sm transition-colors hover:bg-muted"
              >
                <div className="font-medium text-accent">{s.title}</div>
                {s.excerpt && (
                  <div className="mt-0.5 text-xs text-muted-foreground line-clamp-2">
                    {s.excerpt}
                  </div>
                )}
              </a>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
