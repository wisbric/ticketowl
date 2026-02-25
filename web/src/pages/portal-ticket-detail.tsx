import { useState, type FormEvent } from "react";
import { useParams } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { formatRelativeTime } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { SLABadge } from "@/components/tickets/sla-badge";
import { ExternalLink, FileText } from "lucide-react";
import type { PortalTicketDetail, PortalArticle } from "@/types/api";

export function PortalTicketDetailPage() {
  const { ticketId } = useParams({ strict: false });
  const id = Number(ticketId);
  const queryClient = useQueryClient();
  const [replyBody, setReplyBody] = useState("");

  const { data: ticket, isLoading } = useQuery({
    queryKey: ["portal-ticket", id],
    queryFn: () => api.get<PortalTicketDetail>(`/portal/tickets/${id}`),
  });

  const { data: articles } = useQuery({
    queryKey: ["portal-ticket-articles", id],
    queryFn: () => api.get<PortalArticle[]>(`/portal/tickets/${id}/articles`),
  });

  useTitle(ticket ? `#${ticket.number} ${ticket.title}` : "Ticket");

  const replyMutation = useMutation({
    mutationFn: () =>
      api.post(`/portal/tickets/${id}/reply`, { body: replyBody }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["portal-ticket", id] });
      setReplyBody("");
    },
  });

  function handleReply(e: FormEvent) {
    e.preventDefault();
    if (!replyBody.trim()) return;
    replyMutation.mutate();
  }

  if (isLoading) return <LoadingSpinner />;
  if (!ticket) return <p className="text-muted-foreground">Ticket not found.</p>;

  return (
    <div className="mx-auto max-w-4xl">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">#{ticket.number}</h1>
          <h2 className="text-xl">{ticket.title}</h2>
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Badge variant="secondary" className="capitalize">{ticket.status}</Badge>
          <span className="text-sm font-medium capitalize">{ticket.priority}</span>
          {ticket.sla_state && <SLABadge state={ticket.sla_state} />}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Comments */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Conversation</CardTitle>
            </CardHeader>
            <CardContent>
              {ticket.comments && ticket.comments.length > 0 ? (
                <div className="space-y-3">
                  {ticket.comments.map((c) => (
                    <Card key={c.id} className="p-4">
                      <div className="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
                        <span className="font-medium text-foreground">{c.created_by || c.sender}</span>
                        <span>&middot;</span>
                        <span>{formatRelativeTime(c.created_at)}</span>
                      </div>
                      <div
                        className="prose prose-sm max-w-none text-foreground"
                        dangerouslySetInnerHTML={{ __html: c.body }}
                      />
                    </Card>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No messages yet.</p>
              )}

              <form onSubmit={handleReply} className="mt-4 space-y-2">
                <Textarea
                  placeholder="Write a reply..."
                  value={replyBody}
                  onChange={(e) => setReplyBody(e.target.value)}
                  rows={3}
                />
                <div className="flex justify-end">
                  <Button type="submit" size="sm" disabled={replyMutation.isPending}>
                    {replyMutation.isPending ? "Sending..." : "Reply"}
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>

        {/* Side column */}
        <div className="space-y-4">
          {/* SLA due times */}
          {(ticket.response_due_at || ticket.resolution_due_at) && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">SLA</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                {ticket.response_due_at && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Response due</span>
                    <span className="font-mono">{formatRelativeTime(ticket.response_due_at)}</span>
                  </div>
                )}
                {ticket.resolution_due_at && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Resolution due</span>
                    <span className="font-mono">{formatRelativeTime(ticket.resolution_due_at)}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Linked articles */}
          {((articles && articles.length > 0) || (ticket.linked_articles && ticket.linked_articles.length > 0)) && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="flex items-center gap-2 text-sm">
                  <FileText className="h-4 w-4" />
                  Related Articles
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {(articles ?? ticket.linked_articles ?? []).map((art) => (
                    <li key={art.id} className="text-sm">
                      <a
                        href={`http://localhost:3001/articles/${art.slug}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 text-accent hover:underline"
                      >
                        {art.title} <ExternalLink className="h-3 w-3" />
                      </a>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}

          {/* Metadata */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <span>{formatRelativeTime(ticket.created_at)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Updated</span>
                <span>{formatRelativeTime(ticket.updated_at)}</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
