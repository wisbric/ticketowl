import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { Plus, Trash2 } from "lucide-react";
import type { AutoTicketRule } from "@/types/api";

export function AdminRulesPage() {
  useTitle("Admin — Auto-Ticket Rules");
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [alertGroup, setAlertGroup] = useState("");
  const [minSeverity, setMinSeverity] = useState("");
  const [defaultPriority, setDefaultPriority] = useState("normal");
  const [defaultGroup, setDefaultGroup] = useState("");
  const [titleTemplate, setTitleTemplate] = useState("{{.AlertName}}: {{.Summary}}");

  const { data: rules, isLoading } = useQuery({
    queryKey: ["admin-rules"],
    queryFn: () => api.get<AutoTicketRule[]>("/admin/rules"),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<AutoTicketRule>("/admin/rules", {
        name,
        enabled: true,
        alert_group: alertGroup || undefined,
        min_severity: minSeverity || undefined,
        default_priority: defaultPriority,
        default_group: defaultGroup || undefined,
        title_template: titleTemplate,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-rules"] });
      setShowCreate(false);
      resetForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/rules/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-rules"] });
    },
  });

  function resetForm() {
    setName("");
    setAlertGroup("");
    setMinSeverity("");
    setDefaultPriority("normal");
    setDefaultGroup("");
    setTitleTemplate("{{.AlertName}}: {{.Summary}}");
  }

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate();
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Auto-Ticket Rules</h1>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4" /> Add Rule
        </Button>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : !rules || rules.length === 0 ? (
        <EmptyState title="No rules" description="Create your first auto-ticket rule." />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Rules</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Alert Group</TableHead>
                  <TableHead className="w-16" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className="font-medium">{rule.name}</TableCell>
                    <TableCell>
                      <Badge variant={rule.enabled ? "default" : "secondary"}>
                        {rule.enabled ? "Enabled" : "Disabled"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm capitalize">{rule.default_priority}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{rule.alert_group || "Any"}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteMutation.mutate(rule.id)}
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
          <DialogTitle>Create Auto-Ticket Rule</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleCreate}>
          <DialogContent className="space-y-3">
            <div>
              <label className="mb-1 block text-sm font-medium">Name</label>
              <Input value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Default Priority</label>
              <Select value={defaultPriority} onChange={(e) => setDefaultPriority(e.target.value)}>
                <option value="urgent">Urgent</option>
                <option value="high">High</option>
                <option value="normal">Normal</option>
                <option value="low">Low</option>
              </Select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Alert Group (optional)</label>
              <Input value={alertGroup} onChange={(e) => setAlertGroup(e.target.value)} />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Min Severity (optional)</label>
              <Select value={minSeverity} onChange={(e) => setMinSeverity(e.target.value)}>
                <option value="">Any</option>
                <option value="critical">Critical</option>
                <option value="warning">Warning</option>
                <option value="info">Info</option>
              </Select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Default Group (optional)</label>
              <Input value={defaultGroup} onChange={(e) => setDefaultGroup(e.target.value)} />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Title Template</label>
              <Input value={titleTemplate} onChange={(e) => setTitleTemplate(e.target.value)} required />
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
