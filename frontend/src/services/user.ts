import { api, getAccessToken } from "./api";
import type { User } from "../types";

const BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api/v1";

export const userService = {
  getMe: () => api.get<User>("/user/me"),

  updateMe: (data: {
    display_name?: string;
    bio?: string;
    gender?: string;
  }) => api.patch<User>("/user/me", data),

  uploadAvatar: async (file: File): Promise<User> => {
    const formData = new FormData();
    formData.append("avatar", file);
    const token = getAccessToken();
    const response = await fetch(`${BASE_URL}/user/me/avatar`, {
      method: "POST",
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: formData,
    });
    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: { code: "UNKNOWN", message: "Upload failed" } }));
      throw err;
    }
    return response.json();
  },
};
