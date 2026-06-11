import type { ErrorResponse } from "../types";

const BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api/v1";

let accessToken: string | null = null;
let refreshToken: string | null = null;
let onUnauthorized: (() => void) | null = null;

export function setAuthTokens(access: string, refresh: string) {
  accessToken = access;
  refreshToken = refresh;
}

export function clearAuthTokens() {
  accessToken = null;
  refreshToken = null;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function setOnUnauthorized(handler: () => void) {
  onUnauthorized = handler;
}

async function refreshAccessToken(): Promise<boolean> {
  if (!refreshToken) return false;

  try {
    const response = await fetch(`${BASE_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) return false;

    const data = await response.json();
    setAuthTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  options?: { skipAuth?: boolean; rawResponse?: boolean }
): Promise<T> {
  const url = `${BASE_URL}${path}`;
  const headers: Record<string, string> = {};

  if (!options?.rawResponse) {
    headers["Content-Type"] = "application/json";
  }

  if (!options?.skipAuth && accessToken) {
    headers["Authorization"] = `Bearer ${accessToken}`;
  }

  const fetchOptions: RequestInit = {
    method,
    headers,
  };

  if (body && !options?.rawResponse) {
    fetchOptions.body = JSON.stringify(body);
  }

  if (body && options?.rawResponse) {
    fetchOptions.body = body as BodyInit;
  }

  let response = await fetch(url, fetchOptions);

  if (response.status === 401 && !options?.skipAuth && refreshToken) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${accessToken}`;
      response = await fetch(url, fetchOptions);
    } else {
      clearAuthTokens();
      onUnauthorized?.();
      throw new Error("Session expired");
    }
  }

  if (options?.rawResponse) {
    return response as unknown as T;
  }

  if (!response.ok) {
    const errorBody: ErrorResponse = await response.json().catch(() => ({
      error: { code: "UNKNOWN", message: "An unknown error occurred" },
    }));
    throw errorBody;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  patch: <T>(path: string, body?: unknown) => request<T>("PATCH", path, body),
  del: <T>(path: string) => request<T>("DELETE", path),
  upload: <T>(path: string, formData: FormData) =>
    request<T>("POST", path, formData, { rawResponse: false }),
};
