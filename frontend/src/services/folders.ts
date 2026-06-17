import { api } from "./api";
import type { AncestorFolder, Folder } from "../types";

export const folderService = {
  create: (name: string, parentFolderId?: string | null) =>
    api.post<Folder>("/folders", { name, parent_folder_id: parentFolderId || null }),

  list: (parentId?: string) => {
    const query = parentId ? `?parent_id=${parentId}` : "";
    return api.get<{ items: Folder[]; total: number }>(`/folders${query}`);
  },

  get: (folderId: string) => api.get<Folder>(`/folders/${folderId}`),

  getAncestors: (folderId: string) =>
    api.get<AncestorFolder[]>(`/folders/${folderId}/ancestors`),

  update: (folderId: string, data: { name?: string; parent_folder_id?: string | null }) =>
    api.patch<Folder>(`/folders/${folderId}`, data),

  delete: (folderId: string) => api.del<void>(`/folders/${folderId}`),
};
