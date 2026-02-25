import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { ConfigOverview, TestZammadResult } from "@/types/api";

export function AdminZammadPage() {
  useTitle("Admin — Zammad");

  const queryClient = useQueryClient();
  const { data: config } = useQuery({
    queryKey: ["admin-config"],
    queryFn: () => api.get<ConfigOverview>("/admin/config"),
  });

  const [url, setUrl] = useState("");
  const [apiToken, setApiToken] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");
  const [pauseStatuses, setPauseStatuses] = useState("");
  const [prevConfigUrl, setPrevConfigUrl] = useState<string | undefined>(undefined);

  const configUrl = config?.zammad?.url;
  if (configUrl !== prevConfigUrl) {
    setPrevConfigUrl(configUrl);
    if (config?.zammad) {
      setUrl(config.zammad.url);
      setPauseStatuses((config.zammad.pause_statuses ?? []).join(", "));
    }
  }

  const saveMutation = useMutation({
    mutationFn: () =>
      api.put("/admin/config/zammad", {
        url,
        api_token: apiToken,
        webhook_secret: webhookSecret || undefined,
        pause_statuses: pauseStatuses ? pauseStatuses.split(",").map((s) => s.trim()) : undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
    },
  });

  const testMutation = useMutation({
    mutationFn: () =>
      api.post<TestZammadResult>("/admin/config/zammad/test", {
        url,
        api_token: apiToken,
      }),
  });

  function handleSave(e: FormEvent) {
    e.preventDefault();
    saveMutation.mutate();
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-6 text-2xl font-bold">Zammad Configuration</h1>

      <Card>
        <CardHeader>
          <CardTitle>Connection Settings</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label htmlFor="zammad-url" className="mb-1 block text-sm font-medium">URL</label>
              <Input
                id="zammad-url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://zammad.example.com"
                required
              />
            </div>
            <div>
              <label htmlFor="zammad-token" className="mb-1 block text-sm font-medium">API Token</label>
              <Input
                id="zammad-token"
                type="password"
                value={apiToken}
                onChange={(e) => setApiToken(e.target.value)}
                placeholder="Enter Zammad API token"
                required
              />
            </div>
            <div>
              <label htmlFor="webhook-secret" className="mb-1 block text-sm font-medium">Webhook Secret</label>
              <Input
                id="webhook-secret"
                type="password"
                value={webhookSecret}
                onChange={(e) => setWebhookSecret(e.target.value)}
                placeholder="Optional webhook HMAC secret"
              />
            </div>
            <div>
              <label htmlFor="pause-statuses" className="mb-1 block text-sm font-medium">
                SLA Pause Statuses
              </label>
              <Input
                id="pause-statuses"
                value={pauseStatuses}
                onChange={(e) => setPauseStatuses(e.target.value)}
                placeholder="pending close, waiting"
              />
              <p className="mt-1 text-xs text-muted-foreground">Comma-separated Zammad statuses that pause SLA timers.</p>
            </div>

            {saveMutation.isError && (
              <p className="text-sm text-destructive">
                {saveMutation.error instanceof Error ? saveMutation.error.message : "Save failed"}
              </p>
            )}
            {saveMutation.isSuccess && (
              <p className="text-sm text-sla-on-track">Configuration saved.</p>
            )}

            <div className="flex gap-2">
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending ? "Saving..." : "Save"}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => testMutation.mutate()}
                disabled={testMutation.isPending || !url || !apiToken}
              >
                {testMutation.isPending ? "Testing..." : "Test Connection"}
              </Button>
            </div>

            {testMutation.isSuccess && (
              <p className={testMutation.data?.success ? "text-sm text-sla-on-track" : "text-sm text-destructive"}>
                {testMutation.data?.success ? "Connection successful!" : testMutation.data?.error || "Connection failed."}
              </p>
            )}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
