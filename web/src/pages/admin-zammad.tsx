import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { ConfigOverview, TestZammadResult } from "@/types/api";

function ManagedView({ config }: { config: ConfigOverview }) {
  const queryClient = useQueryClient();
  const zammad = config.zammad;

  const [pauseStatuses, setPauseStatuses] = useState(
    (zammad?.pause_statuses ?? []).join(", "),
  );
  const [prevUrl, setPrevUrl] = useState<string | undefined>(undefined);
  if (zammad?.url !== prevUrl) {
    setPrevUrl(zammad?.url);
    setPauseStatuses((zammad?.pause_statuses ?? []).join(", "));
  }

  const testMutation = useMutation({
    mutationFn: () =>
      api.post<TestZammadResult>("/admin/config/zammad/test-stored", {}),
  });

  const pauseMutation = useMutation({
    mutationFn: () =>
      api.put("/admin/config/zammad/pause-statuses", {
        pause_statuses: pauseStatuses
          ? pauseStatuses.split(",").map((s) => s.trim())
          : [],
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
    },
  });

  function handleSavePause(e: FormEvent) {
    e.preventDefault();
    pauseMutation.mutate();
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Connection Status</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-[120px_1fr] gap-y-3 text-sm">
            <span className="text-muted-foreground">URL</span>
            <span className="font-mono">{zammad?.url || "Not configured"}</span>

            <span className="text-muted-foreground">API Token</span>
            <span>
              {zammad?.has_token ? (
                <Badge variant="default">Configured</Badge>
              ) : (
                <Badge variant="destructive">Not configured</Badge>
              )}
            </span>

            <span className="text-muted-foreground">Last Updated</span>
            <span>
              {zammad?.updated_at
                ? new Date(zammad.updated_at).toLocaleString()
                : "Never"}
            </span>
          </div>

          <div className="flex items-center gap-3 pt-2">
            <Button
              variant="outline"
              onClick={() => testMutation.mutate()}
              disabled={testMutation.isPending}
            >
              {testMutation.isPending ? "Testing..." : "Test Connection"}
            </Button>
            {testMutation.isSuccess && (
              <span
                className={
                  testMutation.data?.success
                    ? "text-sm text-sla-on-track"
                    : "text-sm text-destructive"
                }
              >
                {testMutation.data?.success
                  ? "Connection successful!"
                  : testMutation.data?.error || "Connection failed."}
              </span>
            )}
            {testMutation.isError && (
              <span className="text-sm text-destructive">
                {testMutation.error instanceof Error
                  ? testMutation.error.message
                  : "Test failed"}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>SLA Pause Statuses</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSavePause} className="space-y-4">
            <div>
              <Input
                value={pauseStatuses}
                onChange={(e) => setPauseStatuses(e.target.value)}
                placeholder="pending close, waiting"
              />
              <p className="mt-1 text-xs text-muted-foreground">
                Comma-separated Zammad statuses that pause SLA timers.
              </p>
            </div>

            {pauseMutation.isError && (
              <p className="text-sm text-destructive">
                {pauseMutation.error instanceof Error
                  ? pauseMutation.error.message
                  : "Save failed"}
              </p>
            )}
            {pauseMutation.isSuccess && (
              <p className="text-sm text-sla-on-track">Pause statuses saved.</p>
            )}

            <Button type="submit" disabled={pauseMutation.isPending}>
              {pauseMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

function ExternalView({ config }: { config: ConfigOverview }) {
  const queryClient = useQueryClient();

  const [url, setUrl] = useState("");
  const [apiToken, setApiToken] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");
  const [pauseStatuses, setPauseStatuses] = useState("");
  const [prevConfigUrl, setPrevConfigUrl] = useState<string | undefined>(
    undefined,
  );

  const configUrl = config.zammad?.url;
  if (configUrl !== prevConfigUrl) {
    setPrevConfigUrl(configUrl);
    if (config.zammad) {
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
        pause_statuses: pauseStatuses
          ? pauseStatuses.split(",").map((s) => s.trim())
          : undefined,
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
    <Card>
      <CardHeader>
        <CardTitle>Connection Settings</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label
              htmlFor="zammad-url"
              className="mb-1 block text-sm font-medium"
            >
              URL
            </label>
            <Input
              id="zammad-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://zammad.example.com"
              required
            />
          </div>
          <div>
            <label
              htmlFor="zammad-token"
              className="mb-1 block text-sm font-medium"
            >
              API Token
            </label>
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
            <label
              htmlFor="webhook-secret"
              className="mb-1 block text-sm font-medium"
            >
              Webhook Secret
            </label>
            <Input
              id="webhook-secret"
              type="password"
              value={webhookSecret}
              onChange={(e) => setWebhookSecret(e.target.value)}
              placeholder="Optional webhook HMAC secret"
            />
          </div>
          <div>
            <label
              htmlFor="pause-statuses"
              className="mb-1 block text-sm font-medium"
            >
              SLA Pause Statuses
            </label>
            <Input
              id="pause-statuses"
              value={pauseStatuses}
              onChange={(e) => setPauseStatuses(e.target.value)}
              placeholder="pending close, waiting"
            />
            <p className="mt-1 text-xs text-muted-foreground">
              Comma-separated Zammad statuses that pause SLA timers.
            </p>
          </div>

          {saveMutation.isError && (
            <p className="text-sm text-destructive">
              {saveMutation.error instanceof Error
                ? saveMutation.error.message
                : "Save failed"}
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
            <p
              className={
                testMutation.data?.success
                  ? "text-sm text-sla-on-track"
                  : "text-sm text-destructive"
              }
            >
              {testMutation.data?.success
                ? "Connection successful!"
                : testMutation.data?.error || "Connection failed."}
            </p>
          )}
        </form>
      </CardContent>
    </Card>
  );
}

export function AdminZammadPage() {
  useTitle("Admin — Zammad");

  const { data: config } = useQuery({
    queryKey: ["admin-config"],
    queryFn: () => api.get<ConfigOverview>("/admin/config"),
  });

  if (!config) return null;

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-6 text-2xl font-bold">Zammad Integration</h1>
      {config.managed ? (
        <ManagedView config={config} />
      ) : (
        <ExternalView config={config} />
      )}
    </div>
  );
}
