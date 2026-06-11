# NuoNetDisk - System Overview Design Document

> Version 1.0
> Status: Approved

---

## Table of Contents

- [1. Purpose and Scope](#1-purpose-and-scope)
- [2. Design Inputs and Constraints](#2-design-inputs-and-constraints)
- [3. System Module Map](#3-system-module-map)
- [4. Cross-Cutting Design Rules](#4-cross-cutting-design-rules)
- [5. Functional Module Designs](#5-functional-module-designs)
  - [5.1 Authentication and User Management](#51-authentication-and-user-management)
  - [5.2 File Management](#52-file-management)
  - [5.3 Folder Management](#53-folder-management)
  - [5.4 File Sharing](#54-file-sharing)
  - [5.5 Recycle Bin](#55-recycle-bin)
  - [5.6 File Preview - Pending Candidate Only](#56-file-preview---pending-candidate-only)
  - [5.7 Frontend UI Modules](#57-frontend-ui-modules)
  - [5.8 Backend API Modules](#58-backend-api-modules)
- [6. MVP Boundary Summary](#6-mvp-boundary-summary)
- [7. Deferred Decisions and Follow-Ups](#7-deferred-decisions-and-follow-ups)

---

## 1. Purpose and Scope

This document continues the approved requirements and architecture work for NuoNetDisk by defining a system-level overview design from whole system to feature modules.

The design has the following goals:

- Translate approved requirements into implementable functional modules.
- Keep the design aligned with the approved monolithic layered architecture.
- Clarify module goals, submodule boundaries, business flows, dependencies, permissions, and MVP scope.
- Preserve pending decisions without implementing or pre-approving them.
- Provide enough structure for later OpenAPI, backend, frontend, and test work.

This document is not a detailed design. It does not define concrete algorithms, complete data access logic, complete code, migration SQL, or UI pixel-level specifications.

---

## 2. Design Inputs and Constraints

### 2.1 Approved technology stack

The system must use the approved stack only:

| Layer | Approved choice |
|---|---|
| Backend language | Go |
| Backend framework | Gin |
| Frontend | React + TypeScript |
| Build tool | Vite |
| Metadata database | PostgreSQL |
| File object storage | MinIO, S3-compatible |
| Authentication | JWT |
| Architecture style | Backend monolith with frontend-backend separation |

No new runtime dependency, development dependency, microservice, queue, server-side conversion service, or preview engine is introduced by this document.

### 2.2 Approved architectural boundaries

The system follows the approved layered backend structure:

```text
HTTP Handler -> Service -> Repository -> PostgreSQL
                          -> Storage -> MinIO
```

The main responsibility boundaries are:

| Layer | Responsibility |
|---|---|
| Handler | HTTP request parsing, input validation, response formatting, streaming response handoff |
| Service | Business workflow, authorization, coordination across repository and storage |
| Repository | PostgreSQL metadata access only |
| Storage | MinIO object operations only |
| Middleware | Recovery, redacted logging, CORS, rate limiting, JWT authentication |
| Frontend services | Typed API access through native fetch by default |
| Frontend state | React Context + useReducer for MVP |

Handlers must not call repositories or storage directly. Services must not depend on Gin or HTTP request objects.

### 2.3 Data and storage boundary

PostgreSQL stores metadata only. MinIO stores raw file content only.

PostgreSQL owns:

- Users
- Files metadata
- Folders metadata
- Share links
- Refresh token state
- Recycle bin state through deletion fields
- Future permissions or ACL metadata after the folder permission inheritance decision is resolved

MinIO owns:

- Raw file objects only
- Opaque object keys only

User-supplied filenames must never be used as object keys. Object keys must not reveal user identity, folder structure, filenames, or business meaning.

### 2.4 Pending decisions preserved

The following are explicitly not implemented by this overview design:

| Pending decision | Current treatment |
|---|---|
| File preview approach | Candidate only. No preview implementation or conversion service is approved. |
| Folder permission inheritance model | Deferred. MVP uses owner access plus share-link access only. |
| Production infrastructure and CI/CD | Deferred. Local development setup only. |
| Storage quota enforcement | Out of MVP scope. |
| Audit logging and admin panel | Out of MVP scope. |
| File version history | Out of MVP scope. The existing version field is for optimistic locking only. |

---

## 3. System Module Map

At the overview level, NuoNetDisk is organized into the following functional modules:

```text
NuoNetDisk
├── Authentication and User Management
├── File Management
├── Folder Management
├── File Sharing
├── Recycle Bin
├── File Preview (pending candidate only)
├── Frontend UI Modules
└── Backend API Modules
```

The backend modules map to the approved package groups:

| Functional module | Handler | Service | Repository | Storage |
|---|---|---|---|---|
| Authentication and User Management | AuthHandler, UserHandler | AuthService, UserService | UserRepository, RefreshTokenRepository | None |
| File Management | FileHandler | FileService | FileRepository, FolderRepository as needed | FileStorage |
| Folder Management | FolderHandler | FolderService | FolderRepository, FileRepository as needed | None for soft delete, FileStorage for permanent cascade through recycle bin |
| File Sharing | ShareHandler | ShareService | ShareRepository, FileRepository, FolderRepository | FileStorage only when serving allowed file content |
| Recycle Bin | RecycleHandler | RecycleService | FileRepository, FolderRepository | FileStorage for permanent delete |
| File Preview | Not implemented | Not implemented | Not implemented | Not implemented |
| Frontend UI | Pages, components, hooks | Frontend services | API client only | Browser download/stream handling |
| Backend API | Route registration and handlers | All services | All repositories | Storage abstraction |

---

## 4. Cross-Cutting Design Rules

These rules apply to all modules.

### 4.1 Authorization

Every operation on files and folders must authorize in the service layer. Handler-level authentication is not enough.

The MVP authorization model is:

1. Resource owner access: the user identified by JWT owns the file or folder through the resource `user_id`.
2. Share-link access: the request presents a valid share token that is not expired, not revoked, and has the required scope.

No team ACLs, role-based access control, folder-level shared permissions, or inherited permissions are implemented in MVP.

### 4.2 Sensitive data handling

The system must never log:

- Access tokens
- Refresh tokens
- Share tokens
- Passwords
- MinIO credentials
- Raw authorization headers
- Raw URLs containing sensitive route parameters or query strings

Request logs should use sanitized route patterns and non-sensitive metadata only.

### 4.3 Streaming and memory safety

All upload and download flows must stream. The system must not read full file content into memory.

This affects:

- Authenticated file upload
- Authenticated file download
- Shared file access when download is allowed
- Future preview, if approved later

File size limits must be enforced before reading the request body.

### 4.4 Error consistency and resource privacy

Private resource existence must not be leaked through error messages or timing differences.

For unauthorized or inaccessible private files and folders, the service layer should return a generic not-found style error when appropriate. Share-link access failures must always return the same generic 404 response regardless of whether the token is missing, expired, revoked, scope-mismatched, or points to an inaccessible resource.

### 4.5 Partial failure handling

PostgreSQL transactions and MinIO operations are not atomic together. Any flow that writes to both must explicitly handle partial failure.

Examples:

- Upload succeeds in MinIO but metadata insert fails: cleanup the MinIO object.
- Permanent delete metadata handling and MinIO deletion conflict: do not leave database state inconsistent.
- Scheduled cleanup encounters a failed object deletion: log safely and continue processing other items.

### 4.6 API-first workflow

Any new or changed request/response shape must be reflected in `api/openapi.yaml` before handler implementation. This overview design does not modify the OpenAPI contract.

---

## 5. Functional Module Designs

## 5.1 Authentication and User Management

### Module goal

Provide secure account registration, login, token refresh, logout, and current-user profile access for the web application.

This module establishes authenticated identity for protected APIs and supports refresh-token rotation with server-side revocation.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| Registration | Create a new user with email, display name defaults, and hashed password |
| Login | Verify email and password, issue short-lived access token and refresh token |
| Access token validation | Validate JWT access tokens for protected requests |
| Refresh token rotation | Verify refresh token, revoke old token, issue new token pair |
| Replay detection | Detect use of revoked refresh tokens and revoke active sessions for the user |
| Logout | Revoke the submitted refresh token or active refresh tokens as required |
| Current user profile | Return authenticated user's basic profile |

### Core business flows

#### Register

1. Client submits email and password.
2. Handler validates request shape.
3. AuthService normalizes and validates email and password policy.
4. UserRepository checks email uniqueness.
5. AuthService hashes password.
6. UserRepository creates the user metadata row.
7. Handler returns public user data only.

#### Login

1. Client submits email and password.
2. AuthService loads the user by email.
3. AuthService verifies the password hash.
4. AuthService issues a JWT access token.
5. AuthService generates an opaque refresh token and stores only its hash.
6. Handler returns token response.

#### Refresh

1. Client submits refresh token.
2. AuthService parses token identifier and secret.
3. RefreshTokenRepository loads token state.
4. AuthService rejects expired or invalid token.
5. If the token is already revoked, replay handling revokes active refresh tokens for that user.
6. If valid, AuthService revokes the old token and creates a new refresh token.
7. AuthService issues a new access token.
8. Handler returns the new token response.

#### Logout

1. Client submits refresh token while authenticated.
2. AuthService verifies the token belongs to the current user where applicable.
3. RefreshTokenRepository marks the token revoked.
4. Handler returns no content.

#### Get current user

1. JWT middleware authenticates request.
2. UserService loads the current user by ID.
3. Handler returns non-sensitive profile fields.

### Backend service, data table, and API dependencies

| Dependency type | Dependencies |
|---|---|
| Backend services | AuthService, UserService |
| Repositories | UserRepository, RefreshTokenRepository |
| Tables | `users`, `refresh_tokens` |
| APIs | `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, `GET /api/v1/user/me` |
| Middleware | JWT authentication middleware, rate limiting, redacted logging |

### Permission and security requirements

- Passwords must be hashed before storage.
- Refresh tokens must be stored as hashes only.
- Access tokens must be short-lived and validated server-side by middleware.
- Refresh tokens must rotate on every refresh.
- Replay of revoked refresh tokens must revoke active refresh tokens for that user.
- Tokens and passwords must never appear in logs.
- Authentication endpoints should be rate-limited.
- Protected endpoints must return 401 for missing or invalid authentication.
- User profile responses must never include password hashes or refresh token metadata.

### MVP scope

Included in MVP:

- Email/password registration
- Login with JWT access token
- Refresh token issuing, storage, rotation, and revocation
- Logout through refresh token revocation
- Current user profile endpoint
- JWT middleware for protected routes
- Auth endpoint rate limiting

### Non-MVP scope

Excluded from MVP:

- Email verification
- Password reset
- Multi-factor authentication
- Social login or OAuth provider login
- Admin user management
- Organization or tenant membership
- Session management UI beyond basic logout

---

## 5.2 File Management

### Module goal

Provide secure file upload, metadata listing, metadata retrieval, download, rename, move, and soft delete while preserving streaming behavior and strict storage boundaries.

File content is stored only in MinIO. File metadata is stored only in PostgreSQL.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| File listing | List files in a folder with pagination and sorting |
| File metadata retrieval | Return metadata for one accessible file |
| Upload | Stream file content into MinIO and create metadata |
| Download | Stream file content from MinIO through backend |
| Rename | Update file name using optimistic locking |
| Move | Change parent folder using optimistic locking |
| Soft delete | Mark file as deleted and make it visible in recycle bin |
| Metadata validation | Validate names, parent folder, size, MIME detection result, and ownership |
| Storage coordination | Coordinate MinIO object operations and DB metadata consistency |

### Core business flows

#### List files in folder

1. Client requests files for a folder or root.
2. JWT middleware authenticates the user.
3. FileService verifies the folder is accessible to the user if a folder ID is provided.
4. FileRepository returns non-deleted files for the parent folder with pagination and sorting.
5. Handler returns list data and total count.

#### Upload file

1. Client submits multipart upload with optional parent folder.
2. Handler enforces configured size limits before reading the body.
3. FileService verifies the parent folder belongs to the user and is not deleted.
4. FileService sanitizes the display filename and detects MIME type server-side.
5. FileService generates an opaque object key.
6. FileService streams content to MinIO and computes required metadata such as SHA256 as part of the streaming workflow.
7. FileRepository creates the metadata record.
8. If DB metadata creation fails after MinIO upload succeeds, FileService attempts MinIO cleanup.
9. Handler returns file metadata without exposing the object key.

#### Download file

1. Client requests file download.
2. JWT middleware authenticates the user.
3. FileService loads file metadata and verifies ownership.
4. FileService rejects deleted or inaccessible files.
5. FileStorage opens a streaming reader from MinIO.
6. Handler writes a streaming response with safe headers.
7. Internal object key and MinIO URL remain hidden.

#### Rename or move file

1. Client submits new name and/or parent folder with the current version.
2. FileService loads current metadata.
3. FileService verifies ownership and target folder ownership.
4. FileService validates the new name and duplicate constraints.
5. FileRepository updates metadata only if the current version matches.
6. Handler returns updated metadata.

#### Soft delete file

1. Client requests deletion.
2. FileService verifies ownership and that the file is not already deleted.
3. FileRepository sets deletion state and timestamp.
4. MinIO object is not deleted during soft delete.
5. Handler returns no content.

### Backend service, data table, and API dependencies

| Dependency type | Dependencies |
|---|---|
| Backend services | FileService, FolderService or folder authorization helper |
| Repositories | FileRepository, FolderRepository |
| Storage | FileStorage backed by MinIO |
| Tables | `files`, `folders` |
| APIs | `GET /api/v1/files`, `POST /api/v1/files`, `GET /api/v1/files/:id`, `GET /api/v1/files/:id/download`, `PATCH /api/v1/files/:id`, `DELETE /api/v1/files/:id` |
| Middleware | JWT authentication, upload/download rate limiting, redacted logging |

### Permission and security requirements

- Every file operation must authorize ownership or valid share-link access.
- File upload and download must stream and must not buffer full content.
- File size limits must be enforced before reading request content.
- User-supplied filenames must be sanitized and used only as display metadata.
- MinIO object keys must be generated internally and must be opaque.
- Server-side MIME detection is required.
- File metadata responses must not expose object keys or bucket URLs.
- Download must not leak whether another user's private file exists.
- Rename and move require optimistic locking through the version field.
- Path traversal patterns in names or path-like inputs must be rejected or neutralized.
- File operations must not log sensitive request data.

### MVP scope

Included in MVP:

- Upload
- List files by folder with pagination
- Get file metadata
- Download through backend streaming
- Soft delete to recycle bin
- Basic rename and move if included in P1 implementation phase
- Server-side authorization checks
- Server-side MIME detection and SHA256 metadata
- Partial failure cleanup for upload

### Non-MVP scope

Excluded from MVP:

- File preview implementation
- Full-text search
- File version history
- File deduplication behavior based on SHA256
- External S3 storage providers as primary backend
- User storage quota enforcement
- Antivirus scanning
- Server-side content transformation
- Bulk upload optimization beyond basic streaming upload

---

## 5.3 Folder Management

### Module goal

Provide hierarchical folder organization for user-owned files and folders, including creation, listing through file/folder views, rename, move, soft delete cascade, and restore support.

The MVP folder permission model is owner-only. Folder permission inheritance for multi-user ACLs is deferred.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| Folder creation | Create a folder under root or another folder |
| Folder metadata update | Rename or move a folder using optimistic locking |
| Folder hierarchy validation | Verify parent folder exists, belongs to user, and does not create invalid hierarchy |
| Folder listing support | Provide child folders for folder views |
| Folder soft delete cascade | Mark folder subtree and contained files as deleted |
| Folder restore support | Restore folder subtree as much as possible through recycle bin flow |
| Breadcrumb support | Provide enough metadata for frontend navigation |

### Core business flows

#### Create folder

1. Client submits folder name and optional parent folder.
2. JWT middleware authenticates the user.
3. FolderService validates the folder name.
4. FolderService verifies the parent folder belongs to the user and is not deleted.
5. FolderRepository checks duplicate name constraints within the same parent scope.
6. FolderRepository creates folder metadata.
7. Handler returns folder metadata.

#### Rename folder

1. Client submits new name and current version.
2. FolderService loads the folder.
3. FolderService verifies owner access.
4. FolderService validates the name and duplicate constraints.
5. FolderRepository updates metadata only if version matches.
6. Handler returns updated folder metadata.

#### Move folder

1. Client submits target parent folder and current version.
2. FolderService loads source and target folders.
3. FolderService verifies both are owned by the user.
4. FolderService validates that the target is not deleted and does not create an invalid hierarchy.
5. FolderRepository updates parent metadata only if version matches.
6. Handler returns updated folder metadata.

#### Soft delete folder

1. Client requests folder deletion.
2. FolderService verifies ownership.
3. FolderRepository marks the folder, descendant folders, and contained files as deleted in a transaction.
4. MinIO objects are not deleted during soft delete.
5. Handler returns no content.

### Backend service, data table, and API dependencies

| Dependency type | Dependencies |
|---|---|
| Backend services | FolderService, FileService as needed for combined listing behavior |
| Repositories | FolderRepository, FileRepository |
| Storage | None for create, rename, move, and soft delete |
| Tables | `folders`, `files` |
| APIs | `POST /api/v1/folders`, `PATCH /api/v1/folders/:id`, `DELETE /api/v1/folders/:id`, `GET /api/v1/files` for combined folder view support |
| Middleware | JWT authentication, redacted logging |

### Permission and security requirements

- Folder operations must authorize ownership.
- Multi-user folder permissions are not implemented in MVP.
- Folder permission inheritance is a pending design decision and must not be assumed.
- Soft-deleting a folder must cascade deletion state to descendants in PostgreSQL.
- Permanent deletion of folder contents is handled through recycle bin permanent delete flow.
- Rename and move require optimistic locking through the version field.
- Move must prevent invalid self-parenting or moving into a descendant.
- Folder names must be sanitized and must not be treated as filesystem paths.
- Error responses must avoid leaking existence of another user's folders.

### MVP scope

Included in MVP:

- Create folders
- List folders as part of folder browsing
- Rename folders if included in P1 implementation phase
- Move folders if included in P1 implementation phase
- Soft delete folders with cascade deletion state
- Restore support through recycle bin
- Owner-only authorization

### Non-MVP scope

Excluded from MVP:

- Team folder permissions
- Folder ACLs
- Folder permission inheritance
- Organization-wide shared folders
- Quota policies by folder
- Folder-level audit logs
- Custom folder icons or advanced UI metadata

---

## 5.4 File Sharing

### Module goal

Allow resource owners to create, revoke, and use expirable scoped share links for files or folders without exposing private storage implementation details.

Share links provide public access through token validation, not JWT authentication.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| Share link creation | Create a scoped token for a file or folder |
| Share link metadata | Store owner, resource, permission scope, expiry, and revocation state |
| Share access validation | Validate token, expiry, revocation, scope, and resource accessibility |
| Share revocation | Allow owners to revoke share links |
| Shared file response | Serve allowed metadata or download stream depending on scope |
| Shared folder response | Serve allowed folder listing based on token scope |
| Shared UI view | Render public shared resource page without authenticated app shell |

### Core business flows

#### Create share link

1. Authenticated owner selects a file or folder.
2. Client submits resource type, resource ID, permission scope, and expiration request.
3. ShareService verifies the current user owns the resource and it is not deleted.
4. ShareService creates a high-entropy opaque share token.
5. ShareRepository stores token metadata.
6. Handler returns share link metadata and the public URL.
7. Response does not expose MinIO keys or bucket URLs.

#### Revoke share link

1. Authenticated owner requests share link revocation.
2. ShareService loads the share link.
3. ShareService verifies the current user owns the share link.
4. ShareRepository marks it revoked.
5. Handler returns no content.

#### Access shared resource

1. Public user opens `/s/:token` in the frontend or calls share API token route.
2. Backend ShareHandler receives token without JWT authentication.
3. ShareService validates token existence, expiry, revocation state, scope, and resource state.
4. If any validation fails, the same generic 404 response is returned.
5. If valid, backend returns allowed resource metadata, folder listing, or allowed file content stream depending on resource type and permission.
6. Internal object keys and bucket URLs are never returned.

### Backend service, data table, and API dependencies

| Dependency type | Dependencies |
|---|---|
| Backend services | ShareService, FileService or file authorization helper, FolderService or folder authorization helper |
| Repositories | ShareRepository, FileRepository, FolderRepository |
| Storage | FileStorage only when serving allowed file content |
| Tables | `share_links`, `files`, `folders` |
| APIs | `POST /api/v1/shares`, `GET /api/v1/shares/:token`, `DELETE /api/v1/shares/:id` |
| Frontend routes | `/s/:token` |
| Middleware | Public endpoint whitelist for token access, redacted logging, rate limiting |

### Permission and security requirements

- Only the resource owner can create or revoke share links.
- Share tokens must be opaque and high entropy.
- Share tokens must never appear in logs.
- Public share access must validate token state server-side.
- All share access failures must return the same generic 404 response.
- Share permission scope must be enforced server-side.
- Preview-scope sharing must not be treated as copy protection.
- Download-scope sharing must use backend-mediated streaming and must not expose MinIO URLs.
- Deleted resources must not be accessible through share links.
- Folder share access must not leak inaccessible descendants.

### MVP scope

Included in MVP or initial P1 phase:

- Create share links for individual files and folders
- Revoke share links
- Expiration support
- Scope metadata with `preview` and `download`
- Public share access endpoint with token validation
- Generic failure responses for all invalid share access states
- Backend-mediated shared download when download scope is allowed

### Non-MVP scope

Excluded from MVP:

- Password-protected share links
- Share recipient identity tracking
- Public upload drop boxes
- Editable shares
- Per-recipient share permissions
- Share analytics
- Server-side preview conversion
- Guaranteed prevention of content saving for preview-scope shares

---

## 5.5 Recycle Bin

### Module goal

Provide safe deletion behavior through soft delete, user-visible recycle bin listing, restore, permanent delete, and scheduled cleanup of expired deleted items.

Soft delete is the default delete behavior. Permanent delete must be explicit and must remove MinIO objects for files.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| Soft delete integration | File and folder delete operations mark records deleted |
| Recycle bin listing | List deleted files and folders for the owner |
| Restore | Restore deleted items and preserve original hierarchy as much as possible |
| Permanent file delete | Remove MinIO object and file metadata safely |
| Permanent folder delete | Cascade permanent deletion through folder subtree |
| Cleanup task | Periodically remove expired recycle bin items |
| Failure handling | Handle MinIO deletion failure without corrupting DB state |

### Core business flows

#### List recycle bin

1. Client requests recycle bin items.
2. JWT middleware authenticates the user.
3. RecycleService queries deleted files and folders owned by the user.
4. Repository returns paginated deleted resources.
5. Handler returns list data.

#### Restore item

1. Client selects deleted file or folder to restore.
2. RecycleService verifies ownership.
3. RecycleService checks whether original parent hierarchy can be restored.
4. Repository clears deletion state for the selected item and required subtree when restoring a folder.
5. If the original parent no longer exists or remains deleted, service applies the approved fallback behavior from architecture, preserving structure as much as possible.
6. Handler returns restored metadata.

#### Permanent delete file

1. Client confirms permanent deletion.
2. RecycleService verifies ownership and deleted state.
3. RecycleService removes the MinIO object or safely handles already-missing object behavior.
4. Repository removes the metadata record only when DB consistency can be preserved.
5. Handler returns no content or a safe error if cleanup cannot be completed consistently.

#### Permanent delete folder

1. Client confirms permanent deletion of a folder.
2. RecycleService verifies ownership and deleted state.
3. Repository identifies deleted descendant files and folders in the subtree.
4. RecycleService coordinates MinIO deletion for file objects.
5. Repository removes metadata records according to safe consistency rules.
6. Failures are logged without sensitive object details and reported safely.

#### Scheduled cleanup

1. Configured cleanup mechanism finds expired deleted resources.
2. Cleanup uses the same service-layer permanent delete rules.
3. Partial MinIO failures are logged and processing continues for other items.
4. Cleanup must not bypass authorization and consistency rules; system-owned cleanup acts on eligible deleted records only.

### Backend service, data table, and API dependencies

| Dependency type | Dependencies |
|---|---|
| Backend services | RecycleService, FileService and FolderService delete/restore helpers |
| Repositories | FileRepository, FolderRepository |
| Storage | FileStorage for MinIO object deletion |
| Tables | `files`, `folders` |
| APIs | `GET /api/v1/recycle-bin`, `POST /api/v1/recycle-bin/:id/restore`, `DELETE /api/v1/recycle-bin/:id` |
| Scheduled work | Configurable recycle bin cleanup mechanism |
| Middleware | JWT authentication for user requests, redacted logging |

### Permission and security requirements

- Recycle bin operations require authorization.
- Users can only view, restore, or permanently delete their own deleted resources.
- Soft delete does not delete MinIO content.
- Permanent delete requires explicit user confirmation in UI.
- Permanent delete must not expose MinIO object keys to the frontend.
- Folder soft delete must cascade deletion state in PostgreSQL.
- Folder permanent delete must cascade MinIO object deletion for contained files.
- Cleanup logs must not include secrets or sensitive tokens.
- Restore must not create unauthorized access or cross-user parent relationships.

### MVP scope

Included in MVP:

- Soft delete files and folders
- Recycle bin listing
- Restore files and folders
- Permanent delete with explicit confirmation
- Folder cascade behavior for soft delete and permanent delete
- Scheduled cleanup as a configurable P1 mechanism if included in the implementation phase
- Graceful handling of MinIO delete failures

### Non-MVP scope

Excluded from MVP:

- User-configurable retention policy
- Admin-managed recycle bin
- Cross-user restore
- Audit history for deletion events
- Storage quota recovery reporting
- Complex conflict resolution UI for restore name collisions beyond basic safe fallback

---

## 5.6 File Preview - Pending Candidate Only

### Module goal

Record the current candidate direction for future preview support while making clear that no preview implementation is approved in this phase.

The candidate approach is browser-native preview for formats the browser can render inline. This remains pending and must not be implemented until the file preview decision is resolved.

### Submodule breakdown

| Submodule | Current status |
|---|---|
| Browser-native image preview | Candidate only |
| Browser-native PDF preview | Candidate only |
| Plain text preview | Candidate only |
| HTTP range support for preview | Candidate only |
| Audio and video playback | Deferred |
| Server-side conversion | Not approved |
| OCR or document conversion | Not approved |
| Preview UI components | Placeholder or disabled state only |

### Candidate business flow

This flow is for future discussion only and is not approved for implementation:

1. User requests preview for a supported browser-native type.
2. Server authorizes access.
3. Server streams content or partial content without loading the full file.
4. Browser renders inline using native capabilities.
5. Unsupported types show a download-oriented or unsupported-preview message.

### Backend service, data table, and API dependencies

| Dependency type | Current treatment |
|---|---|
| Backend services | None implemented |
| Repositories | None added |
| Storage | None added for preview |
| Tables | No new table |
| APIs | No preview API is implemented or added by this document |
| Frontend routes | No dedicated preview route is implemented by this document |

### Permission and security requirements

If preview is approved later:

- Preview must authorize exactly like download.
- Preview must not load entire files into memory.
- Preview should use streaming and HTTP range behavior where required.
- Preview must not expose MinIO object keys or bucket URLs.
- Preview must not use server-side conversion without explicit approval.
- Preview must not create a new public endpoint except as approved through OpenAPI.
- Preview code must receive dedicated tests because it is a high-risk area.

### MVP scope

Included in MVP:

- No functional preview implementation.
- UI may show disabled, pending, or unsupported preview messaging.
- Share-link scope may retain `preview` as metadata, but actual browser rendering remains pending until approved.

### Non-MVP scope

Excluded from MVP:

- Preview endpoints
- Browser-native preview implementation
- Server-side conversion
- LibreOffice integration
- ffmpeg integration
- OCR
- Audio and video playback
- Thumbnail generation
- Search indexing from file contents

---

## 5.7 Frontend UI Modules

### Module goal

Provide a responsive React + TypeScript single-page application for authentication, file browsing, upload progress, folder operations, sharing, recycle bin workflows, and public share access.

The frontend must remain a client of the backend API. It must not perform authorization decisions that replace server-side checks.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| App shell | Overall layout, navigation, sidebar, header, content area |
| Auth pages | Login and registration forms |
| Protected routing | Redirect unauthenticated users away from protected pages |
| Dashboard page | Current folder view with files, folders, toolbar, breadcrumbs, pagination |
| File list components | Table or grid display of file and folder items |
| Folder tree or navigation | Navigate folder hierarchy and root |
| Upload UI | Upload button, progress display, optional drag-and-drop later |
| Item actions | Rename, move, delete, download, share |
| Share dialog | Create and display share links, select scope and expiration |
| Recycle bin page | List, restore, and permanently delete deleted items |
| Shared page | Public page for `/s/:token` access |
| UI feedback | User-friendly errors, loading states, confirmations, notifications |
| API client layer | Native fetch wrapper, token handling, typed request and response helpers |
| State providers | AuthContext, FileContext, UIContext with useReducer |

### Core business flows

#### Login and session flow

1. User opens login page.
2. Auth page submits credentials through Auth API service.
3. AuthContext stores session state according to approved token storage approach.
4. Protected routes allow dashboard access.
5. API client attaches access token to protected requests.
6. On 401 from an authenticated request, API client attempts refresh where appropriate.
7. Logout clears client state and calls backend logout.

#### Folder browsing flow

1. Dashboard loads current folder route or root.
2. FileContext requests folder contents through API services.
3. UI renders folders, files, breadcrumbs, pagination, and item actions.
4. User navigates into a folder by route change.
5. FileContext reloads contents for the new folder.

#### Upload flow

1. User selects a file through input or approved upload UI.
2. Upload hook creates multipart request.
3. Upload progress is shown in UI.
4. Backend returns metadata after upload completes.
5. FileContext refreshes or inserts the new item in the current folder.
6. Errors are shown without internal details.

#### Delete and recycle bin flow

1. User selects delete on a file or folder.
2. UI asks for confirmation for normal delete.
3. API calls soft-delete endpoint.
4. Current folder view removes the item.
5. Recycle bin page can list the item.
6. User may restore or permanently delete from recycle bin.
7. Permanent delete requires explicit confirmation.

#### Share flow

1. User selects share on a file or folder.
2. Share dialog asks for scope and expiration.
3. API creates share link.
4. UI displays the public URL.
5. Owner may revoke existing share links where supported.
6. Public recipient opens `/s/:token` and sees the shared page.

### Backend service, data table, and API dependencies

| Dependency type | Dependencies |
|---|---|
| Frontend services | `auth`, `files`, `folders`, `shares`, `recycleBin`, base `api` client |
| Backend APIs | All approved `/api/v1` endpoints |
| Backend services | Indirect through API only |
| Data tables | No direct frontend dependency; frontend uses API models only |
| Routes | `/login`, `/register`, `/`, `/folder/:folderId`, `/recycle-bin`, `/s/:token`, catch-all not found |

### Permission and security requirements

- Frontend must treat server as the authority for authentication and authorization.
- Frontend must not expose MinIO keys, bucket URLs, tokens in logs, or sensitive errors.
- Frontend must not rely on hidden buttons as a security control.
- Public share page must not require JWT.
- Protected pages must require client-side route guard for usability, but backend remains authoritative.
- Error messages must be user-friendly and must not reveal internal implementation details.
- Token storage must follow the approved architecture and must avoid unnecessary exposure.
- File download must go through backend endpoints, not MinIO direct URLs.
- Preview UI must remain disabled or pending until approved.

### MVP scope

Included in MVP:

- Login and registration pages
- Protected dashboard page
- Root and folder browsing
- File upload with progress
- File download action
- Create folder
- Delete to recycle bin
- Recycle bin list, restore, and permanent delete
- Basic share link creation and public shared page
- User-friendly error and loading states
- React Context + useReducer state management
- Native fetch API client

### Non-MVP scope

Excluded from MVP:

- Full design system dependency
- External state management library
- Advanced drag-and-drop if not prioritized in P1/P2
- File preview rendering
- Admin UI
- Audit log UI
- Quota UI
- Mobile native app
- Real-time collaboration or WebSocket updates
- Advanced search

---

## 5.8 Backend API Modules

### Module goal

Expose stable, versioned, OpenAPI-defined HTTP APIs for all approved backend capabilities while preserving layered architecture, server-side authorization, streaming, and safe error behavior.

### Submodule breakdown

| Submodule | Responsibility |
|---|---|
| Route registration | Register `/api/v1` routes and public endpoint whitelist |
| Auth API | Registration, login, refresh, logout |
| User API | Current user profile |
| File API | List, upload, metadata, download, update, soft delete |
| Folder API | Create, update, soft delete |
| Share API | Create, access by token, revoke |
| Recycle Bin API | List, restore, permanent delete |
| Middleware module | Recovery, redacted logging, CORS, rate limiting, JWT auth |
| Response module | Consistent success and error responses |
| OpenAPI contract | Source of truth for request and response shapes |

### Core business flows

#### Protected API request flow

1. Request enters Gin router.
2. Recovery and redacted logging middleware wrap request handling.
3. CORS and rate limiting are applied.
4. JWT middleware validates access token unless route is explicitly public.
5. Handler parses request and calls service.
6. Service enforces authorization and business rules.
7. Repository and storage are called only through service orchestration.
8. Handler returns JSON or streaming response.

#### Public share request flow

1. Request enters public share endpoint.
2. Logging redacts token-bearing path information.
3. JWT middleware skips the whitelisted route.
4. ShareHandler passes token to ShareService.
5. ShareService validates token and scope.
6. Handler returns allowed public response or the generic 404 error.

#### Streaming response flow

1. Service verifies authorization and metadata state.
2. Service obtains a streaming reader from storage.
3. Handler writes response headers.
4. Handler streams content to client.
5. No layer loads full content into memory.

### Backend service, data table, and API dependencies

| API module | Services | Tables | Storage |
|---|---|---|---|
| Auth API | AuthService | `users`, `refresh_tokens` | None |
| User API | UserService | `users` | None |
| File API | FileService | `files`, `folders` | MinIO through FileStorage |
| Folder API | FolderService | `folders`, `files` | None for soft delete |
| Share API | ShareService | `share_links`, `files`, `folders` | MinIO only for allowed file stream |
| Recycle Bin API | RecycleService | `files`, `folders` | MinIO for permanent delete |
| Preview API | Not implemented | None | None |

### Permission and security requirements

- All `/api/v1` endpoints require JWT except register, login, refresh, and share token access.
- Refresh validates refresh token state and does not require access JWT.
- Share access validates token state and does not require access JWT.
- Handlers must not bypass services.
- Services must perform authorization for resource operations.
- All request and response shapes must match OpenAPI.
- Error responses must use the approved error schema.
- Download responses must stream.
- Uploads must enforce size limits before body read.
- Middleware must redact tokens and sensitive data.
- CORS must be restrictive in production.
- Rate limits must be configurable for auth, upload, and download endpoints.

### MVP scope

Included in MVP:

- Core `/api/v1` route group
- Public endpoint whitelist
- Auth, user, file, folder, share, and recycle bin handlers
- JSON response helpers
- Streaming download support
- Multipart upload support
- JWT middleware
- Redacted logging
- Rate limiting where required by priority
- OpenAPI sync before handler implementation

### Non-MVP scope

Excluded from MVP:

- Preview API
- Admin API
- Audit API
- Quota API
- Full-text search API
- File versioning API
- WebSocket API
- Microservice boundaries
- Direct MinIO pre-signed URL API unless explicitly approved later

---

## 6. MVP Boundary Summary

### 6.1 MVP included capabilities

| Area | MVP capability |
|---|---|
| Authentication | Register, login, refresh, logout, current user |
| File management | Upload, list, metadata, download, soft delete |
| Folder management | Create, browse, soft delete cascade; rename and move as P1 if scheduled |
| Sharing | Create, revoke, expire, and access scoped share links |
| Recycle bin | List, restore, permanent delete |
| Frontend | Auth pages, dashboard, upload progress, folder browsing, recycle bin, share page |
| Backend | Versioned REST APIs, layered services, repository and storage abstraction |
| Security | Server-side authorization, redacted logs, private-resource-safe errors |
| Reliability | Explicit MinIO and PostgreSQL partial failure handling |

### 6.2 MVP excluded capabilities

| Area | Excluded capability |
|---|---|
| Preview | No functional preview implementation |
| Permissions | No team ACLs or folder permission inheritance |
| Admin | No admin panel or admin API |
| Infrastructure | No production CI/CD or deployment automation beyond basic local development |
| Search | No full-text search |
| Versioning | No file version history |
| Quota | No storage quota enforcement |
| Collaboration | No real-time collaboration |
| Mobile | No native mobile applications |
| Conversion | No LibreOffice, ffmpeg, OCR, thumbnails, or server-side conversion |

---

## 7. Deferred Decisions and Follow-Ups

### 7.1 File preview decision

Current status: pending.

Required before implementation:

- Approve browser-native preview or another approach.
- Define exact supported MIME types.
- Define whether HTTP range support is required for preview endpoints.
- Update OpenAPI before adding preview routes.
- Add tests for authorization, streaming behavior, and memory safety.

No preview code, conversion service, OCR, ffmpeg, LibreOffice, or thumbnail generation should be added before this decision is resolved.

### 7.2 Folder permission inheritance decision

Current status: pending.

Required before implementation:

- Decide inheritance strategy.
- Decide future ACL table shape.
- Define precedence between owner access, inherited access, explicit access, and share-link access.
- Define how folder moves affect inherited permissions.
- Update architecture and OpenAPI as needed before implementation.

MVP must remain owner-only plus share-link access.

### 7.3 Dependency approval follow-up

Current status: no new dependencies introduced by this document.

Before implementation, individual Go modules and npm packages still require explicit approval, even if they support the approved stack.

Examples requiring approval include:

- PostgreSQL driver
- MinIO SDK
- JWT library
- Password hashing package
- Migration runner
- Frontend linting or test packages

### 7.4 Verification follow-up

This document changes documentation only.

Expected verification for this document:

- Confirm the document contains English content only.
- Confirm it does not modify AGENTS.md, requirements.md, or architecture.md.
- Confirm it does not introduce a new technology stack, dependency, microservice, or preview implementation.
- Confirm it does not include migration SQL, detailed algorithms, or complete code.
