import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { formatRelativeTime, priorityColor } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { SLABadge } from "@/components/tickets/sla-badge";
import { Plus } from "lucide-react";
import type { EnrichedTicket, CreateTicketRequest, TicketMetadata } from "@/types/api";

export function TicketListPage() {
  useTitle("The Perch");
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState("");
  const [priorityFilter, setPriorityFilter] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [title, setTitle] = useState("");
  const [groupId, setGroupId] = useState("");
  const [priority, setPriority] = useState("");
  const [body, setBody] = useState("");

  const { data: tickets, isLoading } = useQuery({
    queryKey: ["tickets"],
    queryFn: () => api.get<EnrichedTicket[]>("/tickets"),
  });

  const { data: metadata } = useQuery({
    queryKey: ["ticket-metadata"],
    queryFn: () => api.get<TicketMetadata>("/tickets/metadata"),
    staleTime: 5 * 60 * 1000,
  });

  const createMutation = useMutation({
    mutationFn: (req: CreateTicketRequest) =>
      api.post<EnrichedTicket>("/tickets", req),
    onSuccess: (ticket) => {
      queryClient.invalidateQueries({ queryKey: ["tickets"] });
      setShowCreate(false);
      setTitle("");
      setGroupId("1");
      setPriority("");
      setBody("");
      navigate({ to: "/tickets/$ticketId", params: { ticketId: String(ticket.id) } });
    },
  });

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate({
      title,
      group_id: Number(groupId),
      priority_id: priority ? Number(priority) : undefined,
      body: body || undefined,
    });
  }

  const filtered = (tickets ?? []).filter((t) => {
    if (statusFilter && t.state.toLowerCase() !== statusFilter) return false;
    if (priorityFilter && t.priority.toLowerCase() !== priorityFilter) return false;
    return true;
  });

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">The Perch</h1>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4" /> New Ticket
        </Button>
      </div>

      <div className="mb-4 flex gap-3">
        <Select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="w-40"
        >
          <option value="">All Statuses</option>
          {(metadata?.states ?? []).map((s) => (
            <option key={s.id} value={s.name.toLowerCase()}>{s.name}</option>
          ))}
        </Select>

        <Select
          value={priorityFilter}
          onChange={(e) => setPriorityFilter(e.target.value)}
          className="w-40"
        >
          <option value="">All Priorities</option>
          {(metadata?.priorities ?? []).map((p) => (
            <option key={p.id} value={p.name.toLowerCase()}>{p.name}</option>
          ))}
        </Select>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : filtered.length === 0 ? (
        <EmptyState
          title="No tickets found"
          description="No tickets match your current filters."
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">#</TableHead>
              <TableHead>Title</TableHead>
              <TableHead className="w-28">Status</TableHead>
              <TableHead className="w-28">Priority</TableHead>
              <TableHead className="w-28">SLA</TableHead>
              <TableHead className="w-32">Owner</TableHead>
              <TableHead className="w-28">Updated</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.map((ticket) => (
              <TableRow
                key={ticket.id}
                className="cursor-pointer"
                onClick={() => navigate({ to: "/tickets/$ticketId", params: { ticketId: String(ticket.id) } })}
              >
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {ticket.number}
                </TableCell>
                <TableCell className="font-medium">{ticket.title}</TableCell>
                <TableCell>
                  <span className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-xs font-medium capitalize">
                    {ticket.state}
                  </span>
                </TableCell>
                <TableCell>
                  <span className={priorityColor(ticket.priority) + " text-xs font-medium capitalize"}>
                    {ticket.priority}
                  </span>
                </TableCell>
                <TableCell>
                  {ticket.sla_policy_id ? (
                    <SLABadge state="on_track" />
                  ) : (
                    <span className="text-xs text-muted-foreground">—</span>
                  )}
                </TableCell>
                <TableCell className="text-sm">
                  {ticket.owner || "Unassigned"}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {formatRelativeTime(ticket.updated_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog open={showCreate} onClose={() => setShowCreate(false)}>
        <DialogHeader>
          <DialogTitle>Create Ticket</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleCreate}>
          <DialogContent className="space-y-3">
            <div>
              <label className="mb-1 block text-sm font-medium">Title</label>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Brief summary of the issue"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Group</label>
              <Select value={groupId} onChange={(e) => setGroupId(e.target.value)} required>
                <option value="">Select group...</option>
                {(metadata?.groups ?? []).map((g) => (
                  <option key={g.id} value={String(g.id)}>{g.name}</option>
                ))}
              </Select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Priority</label>
              <Select value={priority} onChange={(e) => setPriority(e.target.value)}>
                <option value="">Default</option>
                {(metadata?.priorities ?? []).map((p) => (
                  <option key={p.id} value={String(p.id)}>{p.name}</option>
                ))}
              </Select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Description</label>
              <Textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder="Detailed description (optional)"
                rows={4}
              />
            </div>
            {createMutation.isError && (
              <p className="text-sm text-destructive">
                {createMutation.error instanceof Error ? createMutation.error.message : "Failed to create ticket"}
              </p>
            )}
          </DialogContent>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </Dialog>
    </div>
  );
}
