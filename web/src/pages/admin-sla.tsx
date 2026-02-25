import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { Plus, Trash2 } from "lucide-react";
import type { SLAPolicy } from "@/types/api";

export function AdminSLAPage() {
  useTitle("The Watch");
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [priority, setPriority] = useState("normal");
  const [responseMinutes, setResponseMinutes] = useState("60");
  const [resolutionMinutes, setResolutionMinutes] = useState("480");
  const [warningThreshold, setWarningThreshold] = useState("0.2");
  const [isDefault, setIsDefault] = useState(false);

  const { data: policies, isLoading } = useQuery({
    queryKey: ["sla-policies"],
    queryFn: () => api.get<SLAPolicy[]>("/sla/policies"),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<SLAPolicy>("/sla/policies", {
        name,
        priority,
        response_minutes: Number(responseMinutes),
        resolution_minutes: Number(resolutionMinutes),
        warning_threshold: Number(warningThreshold),
        is_default: isDefault,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sla-policies"] });
      setShowCreate(false);
      resetForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/sla/policies/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sla-policies"] });
    },
  });

  function resetForm() {
    setName("");
    setPriority("normal");
    setResponseMinutes("60");
    setResolutionMinutes("480");
    setWarningThreshold("0.2");
    setIsDefault(false);
  }

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate();
  }

  function formatDuration(minutes: number): string {
    if (minutes < 60) return `${minutes}m`;
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    return m > 0 ? `${h}h ${m}m` : `${h}h`;
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">The Watch — SLA Policies</h1>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4" /> Add Policy
        </Button>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : !policies || policies.length === 0 ? (
        <EmptyState title="No SLA policies" description="Create your first SLA policy." />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Policies</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Response</TableHead>
                  <TableHead>Resolution</TableHead>
                  <TableHead>Default</TableHead>
                  <TableHead className="w-16" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {policies.map((policy) => (
                  <TableRow key={policy.id}>
                    <TableCell className="font-medium">{policy.name}</TableCell>
                    <TableCell className="capitalize">{policy.priority}</TableCell>
                    <TableCell>{formatDuration(policy.response_minutes)}</TableCell>
                    <TableCell>{formatDuration(policy.resolution_minutes)}</TableCell>
                    <TableCell>
                      {policy.is_default && <Badge variant="default">Default</Badge>}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteMutation.mutate(policy.id)}
                        disabled={deleteMutation.isPending}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Dialog open={showCreate} onClose={() => setShowCreate(false)}>
        <DialogHeader>
          <DialogTitle>Create SLA Policy</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleCreate}>
          <DialogContent className="space-y-3">
            <div>
              <label className="mb-1 block text-sm font-medium">Name</label>
              <Input value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Priority</label>
              <Select value={priority} onChange={(e) => setPriority(e.target.value)}>
                <option value="urgent">Urgent</option>
                <option value="high">High</option>
                <option value="normal">Normal</option>
                <option value="low">Low</option>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-sm font-medium">Response (minutes)</label>
                <Input
                  type="number"
                  value={responseMinutes}
                  onChange={(e) => setResponseMinutes(e.target.value)}
                  min="1"
                  required
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium">Resolution (minutes)</label>
                <Input
                  type="number"
                  value={resolutionMinutes}
                  onChange={(e) => setResolutionMinutes(e.target.value)}
                  min="1"
                  required
                />
              </div>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Warning Threshold</label>
              <Input
                type="number"
                step="0.01"
                value={warningThreshold}
                onChange={(e) => setWarningThreshold(e.target.value)}
                min="0"
                max="1"
              />
              <p className="mt-1 text-xs text-muted-foreground">Fraction of time remaining (e.g. 0.2 = warn at 20% left)</p>
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="is-default"
                checked={isDefault}
                onChange={(e) => setIsDefault(e.target.checked)}
                className="rounded"
              />
              <label htmlFor="is-default" className="text-sm font-medium">Default policy</label>
            </div>
            {createMutation.isError && (
              <p className="text-sm text-destructive">
                {createMutation.error instanceof Error ? createMutation.error.message : "Create failed"}
              </p>
            )}
          </DialogContent>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>Cancel</Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </Dialog>
    </div>
  );
}
