import { Link } from "@tanstack/react-router";
import { useTitle } from "@/hooks/use-title";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Server, Link2, Users, Zap } from "lucide-react";

const adminCards = [
  {
    to: "/admin/zammad",
    title: "Zammad",
    description: "Configure Zammad instance URL, API token, and webhook settings.",
    icon: Server,
  },
  {
    to: "/admin/integrations",
    title: "Integrations",
    description: "Configure NightOwl and BookOwl API keys and URLs.",
    icon: Link2,
  },
  {
    to: "/admin/customers",
    title: "Customer Organizations",
    description: "Manage customer organizations and their OIDC group mappings.",
    icon: Users,
  },
  {
    to: "/admin/rules",
    title: "Auto-Ticket Rules",
    description: "Configure rules that automatically create tickets from NightOwl alerts.",
    icon: Zap,
  },
] as const;

export function AdminPage() {
  useTitle("Admin");

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Admin</h1>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {adminCards.map((card) => (
          <Link key={card.to} to={card.to}>
            <Card className="transition-colors hover:bg-muted/50">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <card.icon className="h-5 w-5 text-accent" />
                  {card.title}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{card.description}</p>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
