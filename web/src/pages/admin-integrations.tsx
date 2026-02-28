import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { ConfigOverview } from "@/types/api";

function IntegrationForm({
  service,
  label,
  currentUrl,
  placeholder,
}: {
  service: "nightowl" | "bookowl";
  label: string;
  currentUrl?: string;
  placeholder?: string;
}) {
  const queryClient = useQueryClient();
  const [apiUrl, setApiUrl] = useState(currentUrl ?? "");
  const [apiKey, setApiKey] = useState("");
  const [prevUrl, setPrevUrl] = useState(currentUrl);

  if (currentUrl !== prevUrl) {
    setPrevUrl(currentUrl);
    if (currentUrl) setApiUrl(currentUrl);
  }

  const mutation = useMutation({
    mutationFn: () =>
      api.put(`/admin/config/${service}`, { api_key: apiKey, api_url: apiUrl }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.mutate();
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium">API URL</label>
            <Input
              value={apiUrl}
              onChange={(e) => setApiUrl(e.target.value)}
              placeholder={placeholder || `https://${service}.example.com`}
              required
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">API Key</label>
            <Input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="Enter API key"
              required
            />
          </div>

          {mutation.isError && (
            <p className="text-sm text-destructive">
              {mutation.error instanceof Error ? mutation.error.message : "Save failed"}
            </p>
          )}
          {mutation.isSuccess && (
            <p className="text-sm text-sla-on-track">Configuration saved.</p>
          )}

          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Saving..." : "Save"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

export function AdminIntegrationsPage() {
  useTitle("Admin — Integrations");

  const { data: config } = useQuery({
    queryKey: ["admin-config"],
    queryFn: () => api.get<ConfigOverview>("/admin/config"),
  });

  const { data: authConfig } = useQuery({
    queryKey: ["auth-config"],
    queryFn: async () => {
      const res = await fetch("/auth/config");
      if (!res.ok) return {};
      return res.json();
    },
    staleTime: Infinity,
  });

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-6 text-2xl font-bold">Integrations</h1>

      <div className="space-y-6">
        <IntegrationForm
          service="nightowl"
          label="NightOwl"
          currentUrl={config?.nightowl?.api_url}
          placeholder={authConfig?.nightowl_api_url || "http://owl-nightowl-api:8080"}
        />
        <IntegrationForm
          service="bookowl"
          label="BookOwl"
          currentUrl={config?.bookowl?.api_url}
          placeholder={authConfig?.bookowl_api_url || "http://owl-bookowl-api:8081"}
        />
      </div>
    </div>
  );
}
