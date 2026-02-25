import { Link } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { useTitle } from "@/hooks/use-title";

export function NotFoundPage() {
  useTitle("Not Found");

  return (
    <EmptyState
      title="Page not found"
      description="The page you're looking for doesn't exist."
      action={
        <Link to="/">
          <Button variant="outline">Back to The Perch</Button>
        </Link>
      }
    />
  );
}
