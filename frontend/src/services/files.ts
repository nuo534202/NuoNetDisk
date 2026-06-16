import { api, getDownloadUrl } from "./api";
import type { File as AppFile, PaginatedResponse } from "../types";

export const fileService = {
  list: (params?: { folder_id?: string; offset?: number; limit?: number; sort?: string; order?: string }) => {
    const query = new URLSearchParams();
    if (params?.folder_id) query.set("folder_id", params.folder_id);
    if (params?.offset) query.set("offset", String(params.offset));
    if (params?.limit) query.set("limit", String(params.limit));
    if (params?.sort) query.set("sort", params.sort);
    if (params?.order) query.set("order", params.order);
    const qs = query.toString();
    return api.get<PaginatedResponse<AppFile>>(`/files${qs ? `?${qs}` : ""}`);
  },

  upload: (file: File, parentFolderId?: string) => {
    const formData = new FormData();
    formData.append("file", file);
    if (parentFolderId) formData.append("parent_folder_id", parentFolderId);
    return api.upload<AppFile>("/files", formData);
  },

  get: (fileId: string) => api.get<AppFile>(`/files/${fileId}`),

  downloadUrl: (fileId: string) => getDownloadUrl(`/files/${fileId}/download`),

  update: (fileId: string, data: { name?: string; parent_folder_id?: string | null }) =>
    api.patch<AppFile>(`/files/${fileId}`, data),

  delete: (fileId: string) => api.del<void>(`/files/${fileId}`),
};
