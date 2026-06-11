import { api } from "./api";
import type { File as AppFile, Folder, PaginatedResponse } from "../types";

export const recycleBinService = {
  listFiles: (params?: { offset?: number; limit?: number }) => {
    const query = new URLSearchParams();
    if (params?.offset) query.set("offset", String(params.offset));
    if (params?.limit) query.set("limit", String(params.limit));
    const qs = query.toString();
    return api.get<PaginatedResponse<AppFile>>(`/recycle-bin/files${qs ? `?${qs}` : ""}`);
  },

  listFolders: (params?: { offset?: number; limit?: number }) => {
    const query = new URLSearchParams();
    if (params?.offset) query.set("offset", String(params.offset));
    if (params?.limit) query.set("limit", String(params.limit));
    const qs = query.toString();
    return api.get<PaginatedResponse<Folder>>(`/recycle-bin/folders${qs ? `?${qs}` : ""}`);
  },

  restoreFile: (fileId: string) => api.post<AppFile>(`/recycle-bin/restore/file/${fileId}`),

  restoreFolder: (folderId: string) => api.post<Folder>(`/recycle-bin/restore/folder/${folderId}`),

  permanentDeleteFile: (fileId: string) => api.del<void>(`/recycle-bin/file/${fileId}`),

  permanentDeleteFolder: (folderId: string) => api.del<void>(`/recycle-bin/folder/${folderId}`),
};
