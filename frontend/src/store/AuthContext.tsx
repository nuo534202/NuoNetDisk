import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import type { User } from "../types";
import { authService } from "../services/auth";
import {
  setAuthTokens,
  clearAuthTokens,
  setOnUnauthorized,
} from "../services/api";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const handleUnauthorized = useCallback(() => {
    setUser(null);
    clearAuthTokens();
  }, []);

  useEffect(() => {
    setOnUnauthorized(handleUnauthorized);
  }, [handleUnauthorized]);

  useEffect(() => {
    const token = sessionStorage.getItem("access_token");
    const refresh = sessionStorage.getItem("refresh_token");
    if (token && refresh) {
      setAuthTokens(token, refresh);
      authService
        .getMe()
        .then((u) => setUser(u))
        .catch(() => {
          clearAuthTokens();
          sessionStorage.removeItem("access_token");
          sessionStorage.removeItem("refresh_token");
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const response = await authService.login(email, password);
    setAuthTokens(response.access_token, response.refresh_token);
    sessionStorage.setItem("access_token", response.access_token);
    sessionStorage.setItem("refresh_token", response.refresh_token);
    const me = await authService.getMe();
    setUser(me);
  }, []);

  const register = useCallback(async (email: string, password: string) => {
    await authService.register(email, password);
  }, []);

  const logout = useCallback(async () => {
    try {
      const refresh = sessionStorage.getItem("refresh_token");
      if (refresh) {
        await authService.logout(refresh);
      }
    } catch {
      // Ignore logout errors
    }
    setUser(null);
    clearAuthTokens();
    sessionStorage.removeItem("access_token");
    sessionStorage.removeItem("refresh_token");
  }, []);

  const updateUser = useCallback((updated: User) => {
    setUser(updated);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        register,
        logout,
        updateUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
