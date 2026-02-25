import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import type { IncidentLink } from "@/types/api";

interface LinkIncidentModalProps {
  ticketId: number;
  open: boolean;
  onClose: () => void;
}

export function LinkIncidentModal({ ticketId, open, onClose }: LinkIncidentModalProps) {
  const [incidentId, setIncidentId] = useState("");
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      api.post<IncidentLink>(`/tickets/${ticketId}/links/incident`, {
        incident_id: incidentId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ticket-links", ticketId] });
      setIncidentId("");
      onClose();
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!incidentId.trim()) return;
    mutation.mutate();
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogHeader>
        <DialogTitle>Link NightOwl Incident</DialogTitle>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <label htmlFor="incident-id" className="mb-1 block text-sm font-medium">
            Incident ID
          </label>
          <Input
            id="incident-id"
            value={incidentId}
            onChange={(e) => setIncidentId(e.target.value)}
            placeholder="Enter NightOwl incident UUID"
            required
          />
          {mutation.isError && (
            <p className="mt-2 text-sm text-destructive">
              {mutation.error instanceof Error ? mutation.error.message : "Failed to link incident"}
            </p>
          )}
        </DialogContent>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Linking..." : "Link Incident"}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
