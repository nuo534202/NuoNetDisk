import { api } from "./api";
import type { User, AuthTokenResponse } from "../types";

export const authService = {
  register: (email: string, password: string) =>
    api.post<User>("/auth/register", { email, password }),

  login: (email: string, password: string) =>
    api.post<AuthTokenResponse>("/auth/login", { email, password }),

  refresh: (refreshToken: string) =>
    api.post<AuthTokenResponse>("/auth/refresh", { refresh_token: refreshToken }),

  logout: (refreshToken: string) =>
    api.post<void>("/auth/logout", { refresh_token: refreshToken }),

  getMe: () => api.get<User>("/user/me"),
};
