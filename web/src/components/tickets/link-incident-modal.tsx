import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { IncidentLink } from "@/types/api";
import { Search } from "lucide-react";

interface NightOwlIncident {
  id: string;
  slug: string;
  summary: string;
  severity: string;
  status: string;
}

interface LinkIncidentModalProps {
  ticketId: number;
  open: boolean;
  onClose: () => void;
}

export function LinkIncidentModal({ ticketId, open, onClose }: LinkIncidentModalProps) {
  return (
    <Dialog open={open} onClose={onClose}>
      {open && <LinkIncidentForm ticketId={ticketId} onClose={onClose} />}
    </Dialog>
  );
}

function LinkIncidentForm({ ticketId, onClose }: { ticketId: number; onClose: () => void }) {
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const queryClient = useQueryClient();

  const { data: results, isFetching } = useQuery({
    queryKey: ["search-incidents", search],
    queryFn: () => api.get<NightOwlIncident[]>(`/search/incidents?q=${encodeURIComponent(search)}`),
    enabled: search.length >= 2,
    placeholderData: (prev) => prev,
  });

  const mutation = useMutation({
    mutationFn: () =>
      api.post<IncidentLink>(`/tickets/${ticketId}/links/incident`, {
        incident_id: selectedId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-links", ticketId] });
      onClose();
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!selectedId) return;
    mutation.mutate();
  }

  const severityColor: Record<string, string> = {
    critical: "text-red-400",
    major: "text-orange-400",
    warning: "text-yellow-400",
    info: "text-blue-400",
  };

  return (
    <form onSubmit={handleSubmit}>
      <DialogHeader>
        <DialogTitle>Link NightOwl Incident</DialogTitle>
      </DialogHeader>
      <DialogContent className="space-y-3">
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setSelectedId("");
            }}
            placeholder="Search incidents by title..."
            className="pl-9"
            autoFocus
          />
        </div>
        {search.length >= 2 && (
          <div className="max-h-48 overflow-y-auto rounded border border-border">
            {isFetching && !results?.length && (
              <p className="p-3 text-sm text-muted-foreground">Searching...</p>
            )}
            {results && results.length === 0 && !isFetching && (
              <p className="p-3 text-sm text-muted-foreground">No incidents found</p>
            )}
            {results?.map((inc) => (
              <button
                key={inc.id}
                type="button"
                onClick={() => setSelectedId(inc.id)}
                className={`w-full text-left px-3 py-2 text-sm hover:bg-accent/10 border-b border-border last:border-0 flex items-center justify-between gap-2 ${
                  selectedId === inc.id ? "bg-accent/20" : ""
                }`}
              >
                <span className="truncate">{inc.summary || inc.slug}</span>
                <span className={`text-xs font-medium flex-shrink-0 ${severityColor[inc.severity] ?? "text-muted-foreground"}`}>
                  {inc.severity}
                </span>
              </button>
            ))}
          </div>
        )}
        {selectedId && (
          <p className="text-xs text-muted-foreground">
            Selected: <span className="font-mono">{selectedId.slice(0, 8)}...</span>
          </p>
        )}
        {mutation.isError && (
          <p className="text-sm text-destructive">
            {mutation.error instanceof Error ? mutation.error.message : "Failed to link incident"}
          </p>
        )}
      </DialogContent>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={mutation.isPending || !selectedId}>
          {mutation.isPending ? "Linking..." : "Link Incident"}
        </Button>
      </DialogFooter>
    </form>
  );
}
