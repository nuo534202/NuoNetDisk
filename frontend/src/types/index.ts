export interface User {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  bio: string;
  gender: string;
  user_hash: string;
  created_at: string;
  updated_at: string;
}

export interface File {
  id: string;
  user_id: string;
  name: string;
  size: number;
  mime_type: string;
  sha256_hash: string;
  parent_folder_id: string | null;
  is_deleted: boolean;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
  expires_at: string | null;
}

export interface AncestorFolder {
  id: string;
  name: string;
}

export interface Folder {
  id: string;
  user_id: string;
  name: string;
  parent_folder_id: string | null;
  is_deleted: boolean;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
  expires_at: string | null;
}

export interface ShareLink {
  id: string;
  user_id: string;
  resource_type: "file" | "folder";
  resource_id: string;
  token: string;
  permission: "read" | "write";
  expires_at: string | null;
  is_revoked: boolean;
  created_at: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  offset: number;
  limit: number;
  has_more: boolean;
}

export interface ErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

export interface AuthTokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}
