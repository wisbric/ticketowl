import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";

interface UserInfo {
  id: string;
  email: string;
  display_name: string;
  role: string;
  tenant_slug?: string;
}

interface AuthState {
  user: UserInfo | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface AuthContextValue extends AuthState {
  login: (user: UserInfo) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function getInitialState(): AuthState {
  if (import.meta.env.DEV) {
    return {
      user: { id: "dev", email: "dev@localhost", display_name: "Dev User", role: "admin" },
      isAuthenticated: true,
      isLoading: false,
    };
  }

  return {
    user: null,
    isAuthenticated: false,
    isLoading: true, // check cookie-based session on mount
  };
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(getInitialState);

  useEffect(() => {
    if (import.meta.env.DEV) return;

    // Prod: validate session cookie by calling /auth/me (cookie sent automatically).
    fetch("/auth/me", { credentials: "same-origin" })
      .then((res) => {
        if (!res.ok) throw new Error("no session");
        return res.json();
      })
      .then((user: UserInfo) => {
        setState({ user, isAuthenticated: true, isLoading: false });
      })
      .catch(() => {
        setState({ user: null, isAuthenticated: false, isLoading: false });
      });
  }, []);

  const login = useCallback((user: UserInfo) => {
    setState({ user, isAuthenticated: true, isLoading: false });
  }, []);

  const logout = useCallback(() => {
    setState({ user: null, isAuthenticated: false, isLoading: false });
    // POST to logout endpoint to clear server-side cookie.
    fetch("/auth/logout", {
      method: "POST",
      credentials: "same-origin",
    }).catch(() => {});
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
