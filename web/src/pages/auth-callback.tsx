import { useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { useAuth } from "@/contexts/auth-context";

export function AuthCallbackPage() {
  const { login } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    // After OIDC callback, the session cookie is already set by the server.
    // Validate it by calling /auth/me, then redirect to /.
    fetch("/auth/me", { credentials: "same-origin" })
      .then((res) => {
        if (!res.ok) throw new Error("no session");
        return res.json();
      })
      .then((user) => {
        login(user);
        navigate({ to: "/" });
      })
      .catch(() => {
        navigate({ to: "/login" });
      });
  }, [login, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <LoadingSpinner label="Completing sign in..." />
    </div>
  );
}
