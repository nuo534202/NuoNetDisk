import { api } from "./api";
import type { ShareLink } from "../types";

export const shareService = {
  create: (data: { resource_type: "file" | "folder"; resource_id: string; permission: "read" | "write"; expires_at?: string | null }) =>
    api.post<ShareLink>("/shares", data),

  revoke: (shareId: string) => api.del<void>(`/shares/${shareId}`),

  accessByToken: (token: string) => api.get<unknown>(`/shares/token/${token}`),
};
