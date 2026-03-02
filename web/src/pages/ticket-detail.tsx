import { useState, type FormEvent } from "react";
import { useParams } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { formatRelativeTime } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { ThreadView } from "@/components/tickets/thread-view";
import { SLATimer } from "@/components/tickets/sla-timer";
import { SLABadge } from "@/components/tickets/sla-badge";
import { LinkIncidentModal } from "@/components/tickets/link-incident-modal";
import { LinkArticleModal } from "@/components/tickets/link-article-modal";
import { SuggestionsPanel } from "@/components/tickets/suggestions-panel";
import { Link2, FileText, Plus, ExternalLink } from "lucide-react";
import type {
  EnrichedTicket,
  ThreadEntry,
  TicketLinks,
  SLAState,
  PostMortemResult,
  TicketMetadata,
} from "@/types/api";

export function TicketDetailPage() {
  const { ticketId } = useParams({ strict: false });
  const id = Number(ticketId);
  const queryClient = useQueryClient();

  const [replyBody, setReplyBody] = useState("");
  const [noteBody, setNoteBody] = useState("");
  const [showLinkIncidentModal, setShowLinkIncidentModal] = useState(false);
  const [showLinkArticleModal, setShowLinkArticleModal] = useState(false);

  const { data: ticket, isLoading: ticketLoading } = useQuery({
    queryKey: ["ticket", id],
    queryFn: () => api.get<EnrichedTicket>(`/tickets/${id}`),
  });

  const { data: thread } = useQuery({
    queryKey: ["ticket-thread", id],
    queryFn: () => api.get<ThreadEntry[]>(`/tickets/${id}/comments`),
  });

  const { data: links } = useQuery({
    queryKey: ["ticket-links", id],
    queryFn: () => api.get<TicketLinks>(`/tickets/${id}/links`),
  });

  const { data: sla } = useQuery({
    queryKey: ["ticket-sla", id],
    queryFn: () => api.get<SLAState>(`/tickets/${id}/sla`),
  });

  const { data: metadata } = useQuery({
    queryKey: ["ticket-metadata"],
    queryFn: () => api.get<TicketMetadata>("/tickets/metadata"),
    staleTime: 5 * 60 * 1000,
  });

  const { data: authConfig } = useQuery({
    queryKey: ["auth-config"],
    queryFn: async () => {
      const res = await fetch("/auth/config");
      return res.json();
    },
    staleTime: 5 * 60 * 1000,
  });

  useTitle(ticket ? `#${ticket.number} ${ticket.title}` : "Ticket");

  const updateMutation = useMutation({
    mutationFn: (body: Record<string, number>) =>
      api.patch<EnrichedTicket>(`/tickets/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket", id] });
    },
  });

  const replyMutation = useMutation({
    mutationFn: () =>
      api.post<ThreadEntry>(`/tickets/${id}/comments`, { body: replyBody }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-thread", id] });
      setReplyBody("");
    },
  });

  const noteMutation = useMutation({
    mutationFn: () =>
      api.post<ThreadEntry>(`/tickets/${id}/comments/internal`, { body: noteBody }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-thread", id] });
      setNoteBody("");
    },
  });

  const postmortemMutation = useMutation({
    mutationFn: () => api.post<PostMortemResult>(`/tickets/${id}/postmortem`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-links", id] });
    },
  });

  function handleReply(e: FormEvent) {
    e.preventDefault();
    if (!replyBody.trim()) return;
    replyMutation.mutate();
  }

  function handleNote(e: FormEvent) {
    e.preventDefault();
    if (!noteBody.trim()) return;
    noteMutation.mutate();
  }

  if (ticketLoading) return <LoadingSpinner />;
  if (!ticket) return <p className="text-muted-foreground">Ticket not found.</p>;

  const publicEntries = (thread ?? []).filter((e) => !e.internal);
  const internalEntries = (thread ?? []).filter((e) => e.internal);

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">#{ticket.number}</h1>
          <h2 className="text-xl">{ticket.title}</h2>
        </div>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Select
            value={String(ticket.state_id)}
            onChange={(e) => updateMutation.mutate({ state_id: Number(e.target.value) })}
            className="w-36"
          >
            {(metadata?.states ?? []).map((s) => (
              <option key={s.id} value={String(s.id)}>{s.name}</option>
            ))}
          </Select>
          <Select
            value={String(ticket.priority_id)}
            onChange={(e) => updateMutation.mutate({ priority_id: Number(e.target.value) })}
            className="w-32"
          >
            {(metadata?.priorities ?? []).map((p) => (
              <option key={p.id} value={String(p.id)}>{p.name}</option>
            ))}
          </Select>
          {sla && <SLABadge state={sla.state} />}
          {updateMutation.isPending && (
            <span className="text-xs text-muted-foreground">Saving...</span>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Main column */}
        <div className="space-y-6 lg:col-span-2">
          {/* Thread */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Conversation</CardTitle>
            </CardHeader>
            <CardContent>
              <ThreadView entries={publicEntries} showInternal={false} />

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

          {/* Internal Notes */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Internal Notes</CardTitle>
            </CardHeader>
            <CardContent>
              {internalEntries.length > 0 ? (
                <ThreadView entries={internalEntries} />
              ) : (
                <p className="text-sm text-muted-foreground">No internal notes yet.</p>
              )}

              <form onSubmit={handleNote} className="mt-4 space-y-2">
                <Textarea
                  placeholder="Add an internal note..."
                  value={noteBody}
                  onChange={(e) => setNoteBody(e.target.value)}
                  rows={2}
                />
                <div className="flex justify-end">
                  <Button type="submit" variant="outline" size="sm" disabled={noteMutation.isPending}>
                    {noteMutation.isPending ? "Adding..." : "Add Note"}
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>

        {/* Side column */}
        <div className="space-y-4">
          {/* SLA Timer */}
          {sla && <SLATimer sla={sla} />}

          {/* Linked Incidents */}
          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2 text-sm">
                  <Link2 className="h-4 w-4" />
                  Linked Incidents
                </CardTitle>
                <Button variant="ghost" size="icon" onClick={() => setShowLinkIncidentModal(true)}>
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {links?.incidents && links.incidents.length > 0 ? (
                <ul className="space-y-2">
                  {links.incidents.map((inc) => (
                    <li key={inc.id} className="flex items-center gap-2 text-sm">
                      <a
                        href={`${authConfig?.nightowl_url || ""}/incidents/${inc.incident_id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 text-accent hover:underline"
                      >
                        {inc.incident_slug || inc.incident_id.slice(0, 8)}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs text-muted-foreground">No linked incidents.</p>
              )}
            </CardContent>
          </Card>

          {/* Linked Articles */}
          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2 text-sm">
                  <FileText className="h-4 w-4" />
                  Linked Articles
                </CardTitle>
                <Button variant="ghost" size="icon" onClick={() => setShowLinkArticleModal(true)}>
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {links?.articles && links.articles.length > 0 ? (
                <ul className="space-y-2">
                  {links.articles.map((art) => (
                    <li key={art.id} className="text-sm">
                      <a
                        href={`${authConfig?.bookowl_url || ""}/articles/${art.article_slug}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 text-accent hover:underline"
                      >
                        {art.article_title || art.article_slug}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs text-muted-foreground">No linked articles.</p>
              )}
            </CardContent>
          </Card>

          {/* Suggestions */}
          <SuggestionsPanel ticketId={id} />

          {/* Post-mortem */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Post-Mortem</CardTitle>
            </CardHeader>
            <CardContent>
              {links?.postmortem ? (
                <a
                  href={links.postmortem.postmortem_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-sm text-accent hover:underline"
                >
                  View Post-Mortem <ExternalLink className="h-3 w-3" />
                </a>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => postmortemMutation.mutate()}
                  disabled={postmortemMutation.isPending}
                >
                  {postmortemMutation.isPending ? "Creating..." : "Create Post-Mortem"}
                </Button>
              )}
            </CardContent>
          </Card>

          {/* Metadata */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Owner</span>
                <span>{ticket.owner || "Unassigned"}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Group</span>
                <span>{ticket.group}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <span>{formatRelativeTime(ticket.created_at)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Updated</span>
                <span>{formatRelativeTime(ticket.updated_at)}</span>
              </div>
              {ticket.tags && ticket.tags.length > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Tags</span>
                  <div className="flex flex-wrap gap-1">
                    {ticket.tags.map((tag) => (
                      <Badge key={tag} variant="secondary" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <LinkIncidentModal
        ticketId={id}
        open={showLinkIncidentModal}
        onClose={() => setShowLinkIncidentModal(false)}
      />
      <LinkArticleModal
        ticketId={id}
        open={showLinkArticleModal}
        onClose={() => setShowLinkArticleModal(false)}
      />
    </div>
  );
}
