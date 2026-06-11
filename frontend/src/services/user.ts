import { api } from "./api";
import type { User } from "../types";

export const userService = {
  getMe: () => api.get<User>("/user/me"),

  updateMe: (data: { display_name: string }) =>
    api.patch<User>("/user/me", data),
};
