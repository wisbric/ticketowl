import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { Check, Wifi, Eye, EyeOff, RotateCw, Shield } from "lucide-react";

interface OIDCConfig {
  id: string;
  issuer_url: string;
  client_id: string;
  client_secret: string;
  enabled: boolean;
  tested_at?: string;
  source?: string;
}

interface OIDCTestResult {
  ok: boolean;
  error?: string;
  issuer?: string;
  tested_at?: string;
}

interface ResetResult {
  password: string;
  message: string;
}

interface OIDCForm {
  issuer_url: string;
  client_id: string;
  client_secret: string;
  enabled: boolean;
}

const emptyForm: OIDCForm = {
  issuer_url: "",
  client_id: "",
  client_secret: "",
  enabled: false,
};

export function AdminAuthPage() {
  useTitle("Authentication");
  const queryClient = useQueryClient();

  const [formOverride, setFormOverride] = useState<OIDCForm | null>(null);
  const [saved, setSaved] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [resetPassword, setResetPassword] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-oidc-config"],
    queryFn: () => api.get<OIDCConfig>("/admin/oidc/config"),
  });

  const form = useMemo<OIDCForm>(() => {
    if (formOverride) return formOverride;
    if (!data || !data.id) return {
      issuer_url: data?.issuer_url || "",
      client_id: data?.client_id || "",
      client_secret: data?.client_secret || "",
      enabled: data?.enabled ?? false,
    };
    return {
      issuer_url: data.issuer_url || "",
      client_id: data.client_id || "",
      client_secret: data.client_secret || "",
      enabled: data.enabled ?? false,
    };
  }, [data, formOverride]);

  const setForm = (update: OIDCForm | ((prev: OIDCForm) => OIDCForm)) => {
    setFormOverride(typeof update === "function" ? update(form) : update);
  };

  const saveMutation = useMutation({
    mutationFn: (data: OIDCForm) => api.put<{ status: string }>("/admin/oidc/config", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-oidc-config"] });
      setFormOverride(null);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    },
  });

  const testMutation = useMutation({
    mutationFn: () => api.post<OIDCTestResult>("/admin/oidc/test", {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-oidc-config"] });
    },
  });

  const resetMutation = useMutation({
    mutationFn: () => api.post<ResetResult>("/admin/local-admin/reset", {}),
    onSuccess: (data) => {
      setResetPassword(data.password);
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    saveMutation.mutate(form);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/admin" className="text-muted-foreground hover:text-foreground text-sm">
          &larr; Admin
        </Link>
        <h1 className="text-2xl font-bold">Authentication</h1>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="grid gap-6">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <Shield className="h-5 w-5" />
                  OIDC / Keycloak
                </CardTitle>
                <div className="flex items-center gap-2">
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={form.enabled}
                      onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                      className="accent-accent"
                    />
                    Enabled
                  </label>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {isLoading ? (
                <LoadingSpinner size="sm" />
              ) : (
                <>
                  {data?.source === "environment" && (
                    <div className="rounded-md border border-accent/30 bg-accent/5 px-3 py-2 text-xs text-accent">
                      OIDC is configured via environment variables (Helm/deployment). To override per-tenant, fill in the fields below and save.
                    </div>
                  )}
                  <div>
                    <label className="text-sm font-medium">Issuer URL</label>
                    <Input
                      value={form.issuer_url}
                      onChange={(e) => setForm({ ...form, issuer_url: e.target.value })}
                      placeholder="https://keycloak.example.com/realms/owls"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      The OIDC provider issuer URL (e.g., Keycloak realm URL)
                    </p>
                  </div>

                  <div>
                    <label className="text-sm font-medium">Client ID</label>
                    <Input
                      value={form.client_id}
                      onChange={(e) => setForm({ ...form, client_id: e.target.value })}
                      placeholder="ticketowl"
                    />
                  </div>

                  <div>
                    <label className="text-sm font-medium flex items-center justify-between">
                      Client Secret
                      <button
                        type="button"
                        onClick={() => setShowSecret(!showSecret)}
                        className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
                      >
                        {showSecret ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                        {showSecret ? "Hide" : "Reveal"}
                      </button>
                    </label>
                    <Input
                      type={showSecret ? "text" : "password"}
                      value={form.client_secret}
                      onChange={(e) => setForm({ ...form, client_secret: e.target.value })}
                      placeholder="Enter client secret"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Stored encrypted (AES-256-GCM)
                    </p>
                  </div>

                  <div className="flex items-center gap-3">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => testMutation.mutate()}
                      disabled={testMutation.isPending || (!data?.id && data?.source !== "environment")}
                    >
                      <Wifi className="h-3 w-3 mr-1" />
                      {testMutation.isPending ? "Testing..." : "Test Connection"}
                    </Button>

                    {testMutation.data && (
                      <span
                        className={`text-xs ${testMutation.data.ok ? "text-severity-ok" : "text-destructive"}`}
                      >
                        {testMutation.data.ok
                          ? `Connected${data?.tested_at ? ` · Last tested ${new Date(data.tested_at).toLocaleString()}` : ""}`
                          : `Error: ${testMutation.data.error}`}
                      </span>
                    )}

                    {!testMutation.data && data?.tested_at && (
                      <span className="text-xs text-muted-foreground">
                        Last tested: {new Date(data.tested_at).toLocaleString()}
                      </span>
                    )}
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Local Admin Account</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-6 text-sm">
                <div>
                  <span className="text-muted-foreground">Username:</span>{" "}
                  <span className="font-mono">admin</span>
                </div>
              </div>

              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setResetPassword(null);
                    resetMutation.mutate();
                  }}
                  disabled={resetMutation.isPending}
                >
                  <RotateCw className="h-3 w-3 mr-1" />
                  {resetMutation.isPending ? "Resetting..." : "Reset Password"}
                </Button>

                {resetPassword && (
                  <div className="rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2">
                    <p className="text-xs text-amber-500 mb-1">New password (shown once):</p>
                    <code className="text-sm font-mono font-bold">{resetPassword}</code>
                  </div>
                )}

                {resetMutation.isError && (
                  <p className="text-sm text-destructive">{resetMutation.error.message}</p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="flex items-center gap-3 mt-6">
          <Button type="submit" disabled={saveMutation.isPending || isLoading}>
            {saveMutation.isPending ? "Saving..." : "Save & Reload"}
          </Button>
          {formOverride && (
            <Button
              type="button"
              variant="outline"
              onClick={() => setFormOverride(null)}
            >
              Cancel
            </Button>
          )}
          {saved && (
            <span className="flex items-center gap-1 text-sm text-severity-ok">
              <Check className="h-4 w-4" />
              Configuration saved
            </span>
          )}
          {saveMutation.isError && (
            <p className="text-sm text-destructive">Error: {saveMutation.error.message}</p>
          )}
        </div>
      </form>
    </div>
  );
}
