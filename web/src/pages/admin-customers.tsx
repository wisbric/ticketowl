import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { Plus, Trash2 } from "lucide-react";
import type { CustomerOrg } from "@/types/api";

export function AdminCustomersPage() {
  useTitle("Admin — Customers");
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [oidcGroup, setOidcGroup] = useState("");
  const [zammadOrgId, setZammadOrgId] = useState("");

  const { data: orgs, isLoading } = useQuery({
    queryKey: ["admin-customers"],
    queryFn: () => api.get<CustomerOrg[]>("/admin/customers"),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      api.post<CustomerOrg>("/admin/customers", {
        name,
        oidc_group: oidcGroup,
        zammad_org_id: zammadOrgId ? Number(zammadOrgId) : undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-customers"] });
      setShowCreate(false);
      setName("");
      setOidcGroup("");
      setZammadOrgId("");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/admin/customers/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-customers"] });
    },
  });

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate();
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Customer Organizations</h1>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4" /> Add
        </Button>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : !orgs || orgs.length === 0 ? (
        <EmptyState title="No customer organizations" description="Create your first customer organization." />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Organizations</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>OIDC Group</TableHead>
                  <TableHead className="w-28">Zammad Org</TableHead>
                  <TableHead className="w-16" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {orgs.map((org) => (
                  <TableRow key={org.id}>
                    <TableCell className="font-medium">{org.name}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{org.oidc_group}</TableCell>
                    <TableCell className="text-sm">{org.zammad_org_id ?? "—"}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteMutation.mutate(org.id)}
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
          <DialogTitle>Add Customer Organization</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleCreate}>
          <DialogContent className="space-y-3">
            <div>
              <label className="mb-1 block text-sm font-medium">Name</label>
              <Input value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">OIDC Group</label>
              <Input value={oidcGroup} onChange={(e) => setOidcGroup(e.target.value)} required />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Zammad Org ID (optional)</label>
              <Input value={zammadOrgId} onChange={(e) => setZammadOrgId(e.target.value)} type="number" />
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
