import { api } from "./api";
import type { DashboardData, AdminUser, PaginatedResponse } from "../types";

export const adminService = {
  getDashboard: () => api.get<DashboardData>("/admin/dashboard"),

  listUsers: (params: {
    offset?: number;
    limit?: number;
    q?: string;
    email_filter?: string;
    name_filter?: string;
    role_filter?: string;
    gender_filter?: string;
    date_from?: string;
    date_to?: string;
  } = {}) => {
    const { offset = 0, limit = 20, q = "", email_filter = "", name_filter = "", role_filter = "", gender_filter = "", date_from = "", date_to = "" } = params;
    const query = new URLSearchParams({ offset: String(offset), limit: String(limit) });
    if (q) query.set("q", q);
    if (email_filter) query.set("email_filter", email_filter);
    if (name_filter) query.set("name_filter", name_filter);
    if (role_filter) query.set("role_filter", role_filter);
    if (gender_filter) query.set("gender_filter", gender_filter);
    if (date_from) query.set("date_from", date_from);
    if (date_to) query.set("date_to", date_to);
    return api.get<PaginatedResponse<AdminUser>>(`/admin/users?${query.toString()}`);
  },

  registerAdmin: (email: string, password: string) =>
    api.post<void>("/admin/users/register", { email, password }),
};
