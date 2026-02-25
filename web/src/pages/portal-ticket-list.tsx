import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { formatRelativeTime } from "@/lib/utils";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { SLABadge } from "@/components/tickets/sla-badge";
import type { PortalTicket } from "@/types/api";

export function PortalTicketListPage() {
  useTitle("My Tickets");
  const navigate = useNavigate();

  const { data: tickets, isLoading } = useQuery({
    queryKey: ["portal-tickets"],
    queryFn: () => api.get<PortalTicket[]>("/portal/tickets"),
  });

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">My Tickets</h1>

      {isLoading ? (
        <LoadingSpinner />
      ) : !tickets || tickets.length === 0 ? (
        <EmptyState
          title="No tickets"
          description="You don't have any tickets yet."
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
              <TableHead className="w-28">Updated</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tickets.map((ticket) => (
              <TableRow
                key={ticket.id}
                className="cursor-pointer"
                onClick={() => navigate({ to: "/portal/tickets/$ticketId", params: { ticketId: String(ticket.id) } })}
              >
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {ticket.number}
                </TableCell>
                <TableCell className="font-medium">{ticket.title}</TableCell>
                <TableCell>
                  <span className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-xs font-medium capitalize">
                    {ticket.status}
                  </span>
                </TableCell>
                <TableCell className="text-xs font-medium capitalize">
                  {ticket.priority}
                </TableCell>
                <TableCell>
                  {ticket.sla_state ? (
                    <SLABadge state={ticket.sla_state} />
                  ) : (
                    <span className="text-xs text-muted-foreground">—</span>
                  )}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {formatRelativeTime(ticket.updated_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
