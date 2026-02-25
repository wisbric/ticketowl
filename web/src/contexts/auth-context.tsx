import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";

interface UserInfo {
  id: string;
  email: string;
  display_name: string;
  role: string;
  tenant_slug?: string;
}

interface AuthState {
  token: string | null;
  user: UserInfo | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface AuthContextValue extends AuthState {
  login: (token: string, user: UserInfo) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const TOKEN_KEY = "ticketowl_token";

function getInitialState(): AuthState {
  if (import.meta.env.DEV) {
    return {
      token: null,
      user: { id: "dev", email: "dev@localhost", display_name: "Dev User", role: "admin" },
      isAuthenticated: true,
      isLoading: false,
    };
  }

  const stored = localStorage.getItem(TOKEN_KEY);
  return {
    token: stored,
    user: null,
    isAuthenticated: false,
    isLoading: !!stored,
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

    const stored = localStorage.getItem(TOKEN_KEY);
    if (!stored) return;

    fetch("/auth/me", {
      headers: { Authorization: `Bearer ${stored}` },
    })
      .then((res) => {
        if (!res.ok) throw new Error("invalid token");
        return res.json();
      })
      .then((user: UserInfo) => {
        setState({ token: stored, user, isAuthenticated: true, isLoading: false });
      })
      .catch(() => {
        localStorage.removeItem(TOKEN_KEY);
        setState({ token: null, user: null, isAuthenticated: false, isLoading: false });
      });
  }, []);

  const login = useCallback((token: string, user: UserInfo) => {
    localStorage.setItem(TOKEN_KEY, token);
    setState({ token, user, isAuthenticated: true, isLoading: false });
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    setState({ token: null, user: null, isAuthenticated: false, isLoading: false });
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
