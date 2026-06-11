# NuoNetDisk — Requirements Document

> Version 1.0 — Project Kickoff
> Status: Approved

---

## Table of Contents

- [NuoNetDisk — Requirements Document](#nuonetdisk--requirements-document)
  - [Table of Contents](#table-of-contents)
  - [1. Project Overview](#1-project-overview)
    - [1.1 Project Goals](#11-project-goals)
    - [1.2 Project Status](#12-project-status)
  - [2. Target Users](#2-target-users)
  - [3. Functional Requirements](#3-functional-requirements)
    - [3.1 Authentication \& User Management](#31-authentication--user-management)
    - [3.2 File Management](#32-file-management)
    - [3.3 Folder Management](#33-folder-management)
    - [3.4 File Sharing](#34-file-sharing)
    - [3.5 Recycle Bin](#35-recycle-bin)
    - [3.6 File Preview](#36-file-preview)
  - [4. Non-Functional Requirements](#4-non-functional-requirements)
    - [4.1 Performance](#41-performance)
    - [4.2 Security](#42-security)
    - [4.3 Scalability \& Availability](#43-scalability--availability)
    - [4.4 Reliability \& Data Integrity](#44-reliability--data-integrity)
    - [4.5 Usability](#45-usability)
  - [5. System Architecture](#5-system-architecture)
    - [5.1 High-Level Architecture](#51-high-level-architecture)
    - [5.2 Backend Structure](#52-backend-structure)
    - [5.3 Frontend Structure](#53-frontend-structure)
    - [5.4 API Contract](#54-api-contract)
  - [6. API Design Principles](#6-api-design-principles)
    - [6.1 Core Endpoints (Planned)](#61-core-endpoints-planned)
  - [7. Data Model Overview](#7-data-model-overview)
    - [7.1 Entities (PostgreSQL)](#71-entities-postgresql)
    - [7.2 Storage Boundary Rules](#72-storage-boundary-rules)
  - [8. Constraints](#8-constraints)
    - [8.1 Technology Constraints](#81-technology-constraints)
    - [8.2 Development Constraints](#82-development-constraints)
    - [8.3 Pending Decisions (DO NOT Implement Without Resolution)](#83-pending-decisions-do-not-implement-without-resolution)
    - [8.4 Outright Bans](#84-outright-bans)
  - [9. Out of Scope](#9-out-of-scope)
  - [10. Glossary](#10-glossary)

---

## 1. Project Overview

NuoNetDisk is a personal/team cloud storage application that allows users to upload, manage, share, and organize files through a web interface. It follows a frontend-backend separated architecture with a Go/Gin backend and a React + TypeScript frontend.

The system uses PostgreSQL for metadata storage and MinIO (S3-compatible) for raw file object storage. Authentication is handled via JWT tokens.

### 1.1 Project Goals

- Provide a self-hosted cloud storage solution with a modern web UI.
- Ensure secure file operations with robust permission and access control.
- Support file sharing via expirable, revocable links.
- Implement a recycle bin with soft-delete and restore capabilities.
- Maintain streaming-based file operations to handle large files efficiently.
- Keep the codebase simple, testable, and maintainable.

### 1.2 Project Status

Greenfield project. Repository is empty. Build from scratch with a backend monolith. No microservices.

---

## 2. Target Users

| Persona | Description | Key Needs |
|---|---|---|
| Individual User | A single user managing personal files | Upload, download, organize, basic sharing |
| Team Member | A user within a small team sharing files | Folder organization, share links, permission control (MVP: share-link only, see §3.3 scope note) |
| Administrator (Future) | User with system oversight — **not in MVP scope** | User management, storage quota, audit logs (see §9) |

*Note: Multi-tenant / organization support is not in the initial scope. The Administrator persona is listed here for architectural awareness only — all corresponding features (admin panel, quota enforcement, audit logging) are explicitly out of scope for MVP (see §9).*

---

## 3. Functional Requirements

### 3.1 Authentication & User Management

| ID | Requirement | Priority |
|---|---|---|
| AUTH-01 | Users shall register with email and password. | P0 |
| AUTH-02 | Users shall log in with email and password, receiving a JWT token. | P0 |
| AUTH-03 | JWT tokens shall expire after a configurable duration. | P0 |
| AUTH-04 | Users shall be able to refresh their JWT token using a refresh token. | P1 |
| AUTH-05 | On logout, the server shall revoke the user's active refresh tokens in the database (set `is_revoked`). Access token invalidation is not enforced server-side (short-lived by design); rely on short token expiry. | P1 |
| AUTH-06 | Unauthenticated requests to protected endpoints shall return 401. | P0 |
| AUTH-07 | Passwords shall be hashed (bcrypt or equivalent) before storage. | P0 |
| AUTH-08 | Never log JWT tokens, passwords, or credentials under any circumstance. | P0 |
| AUTH-09 | Refresh tokens must be stored as hashes only (never plaintext). On each refresh, the old token must be revoked immediately and a new token issued (rotation). Replay of a revoked refresh token must be detected and should revoke all active refresh tokens for that user. | P1 |

### 3.2 File Management

| ID | Requirement | Priority |
|---|---|---|
| FILE-01 | Users shall upload files through the web UI. | P0 |
| FILE-02 | Uploads shall stream via `io.Reader`; never load entire file into memory. | P0 |
| FILE-03 | Enforce file size limits during upload before reading the request body. | P0 |
| FILE-04 | User-supplied filenames shall NEVER be used as MinIO object keys. Use UUIDs or generated opaque paths. | P0 |
| FILE-05 | MinIO object keys must be opaque — no user-identifiable or path-revealing strings. | P0 |
| FILE-06 | MIME type shall be detected server-side; do not trust client-supplied MIME types. | P0 |
| FILE-07 | Files shall have a SHA256 hash computed and stored in metadata. | P0 |
| FILE-08 | Users shall download files through the web UI with streaming (`io.Writer`). | P0 |
| FILE-09 | Users shall rename files within a folder. | P1 |
| FILE-10 | Users shall move files between folders. | P1 |
| FILE-11 | Users shall delete files (soft-delete to recycle bin — see §3.5). | P0 |
| FILE-12 | All file operations (upload, download, rename, move, delete, preview, share) must check authorization. | P0 |
| FILE-13 | Prevent path traversal attacks on user-supplied filenames and paths. | P0 |

### 3.3 Folder Management

| ID | Requirement | Priority |
|---|---|---|
| FOLD-01 | Users shall create folders. | P0 |
| FOLD-02 | Users shall rename folders. | P1 |
| FOLD-03 | Users shall delete folders (cascade soft-delete to sub-files and sub-folders). | P0 |
| FOLD-04 | Soft-deleting a folder must cascade to sub-files and sub-folders in DB only (set `deleted_at`). Permanent deletion (see BIN-04) must cascade MinIO object removal. | P0 |
| FOLD-05 | Restore from recycle bin must preserve the original directory structure as much as possible. | P0 |
| FOLD-06 | Folder permission inheritance strategy must be explicitly designed before implementing folder-level permissions. | P2 |

> **MVP scope note on permissions**: For MVP, only two authorization models are implemented: (1) resource owner (the user who created the file/folder has full control), and (2) share-link-based access (see §3.4). Multi-user folder permissions, team ACLs, and role-based access are explicitly excluded from MVP and require the pending permission inheritance decision (§8.3) to be resolved first.

### 3.4 File Sharing

| ID | Requirement | Priority |
|---|---|---|
| SHARE-01 | Users shall generate share links for individual files or folders. | P1 |
| SHARE-02 | Share links shall be revocable (owner can invalidate at any time). | P1 |
| SHARE-03 | Share links shall be expirable (configurable TTL). | P1 |
| SHARE-04 | Share links shall be limited in scope (preview-only UI vs. explicit download endpoint access). **Note**: browser-based preview does not prevent the recipient from saving content — the scope limits the access pattern (in-browser preview vs. allowed download), not absolute copy protection. | P1 |
| SHARE-05 | Shared file access (via `GET /api/v1/shares/:token`) does NOT require JWT authentication. Instead, it enforces server-side access control via share-token validation: check existence, expiry, revocation status, and permission scope. All failure cases (token not found, expired, revoked, mismatched scope, or resource inaccessible) must return the **same generic 404 response** to prevent side-channel leaks about resource existence or state. | P0 |
| SHARE-06 | Do not expose internal MinIO object keys or bucket URLs to the frontend or share links. | P0 |
| SHARE-07 | Unauthorized users must not be able to detect whether a private file exists (no timing or error-message leaks). | P1 |
| SHARE-08 | Share tokens must never appear in logs. | P0 |

### 3.5 Recycle Bin

| ID | Requirement | Priority |
|---|---|---|
| BIN-01 | "Delete" on files/folders shall perform a soft-delete — move to recycle bin, set `deleted_at` timestamp. | P0 |
| BIN-02 | Users shall view items in the recycle bin. | P0 |
| BIN-03 | Users shall restore items from the recycle bin — restoring the original parent structure. | P0 |
| BIN-04 | "Permanent delete" shall remove the MinIO object and the DB record, requiring explicit user confirmation. | P0 |
| BIN-05 | Permanent delete must handle MinIO cleanup failures gracefully (log error, do not leave DB in inconsistent state). | P0 |
| BIN-06 | Recycle bin operations still require permission checks. | P0 |
| BIN-07 | A periodic cleanup process (see NFR-REL-03) shall permanently remove expired recycle bin items and must handle partial MinIO delete failures gracefully (log and continue, do not block subsequent cleanup). | P1 |
| BIN-08 | Deleting a folder must cascade soft-delete to all sub-items. | P0 |

### 3.6 File Preview

| ID | Requirement | Priority |
|---|---|---|
| PREV-01 | Candidate approach: browser-native preview for formats the browser can render inline — images (JPEG, PNG, GIF, WebP, SVG), plain text, and PDF (browser's built-in PDF viewer). Audio/video playback is deferred. **This is a candidate only; not implemented until pending decision (§8.3) is resolved.** | P2 |
| PREV-02 | If/when preview is implemented, it must not load entire files into memory. Stream partial content using HTTP range requests. | P2 |
| PREV-03 | Do NOT introduce server-side conversion services (LibreOffice, ffmpeg, OCR) without explicit approval. Server-side processing of any kind for preview is out of scope until the preview approach decision is resolved. | P0 |

> **Note**: File preview approach is a pending decision (§8.3). The browser-native preview described in PREV-01 is a candidate approach for discussion, not an approved design. Do not implement any preview logic until the decision is resolved.

---

## 4. Non-Functional Requirements

### 4.1 Performance

| ID | Requirement | Priority |
|---|---|---|
| NFR-PERF-01 | Upload and download operations shall stream data — never buffer full content in memory. | P0 |
| NFR-PERF-02 | Benchmark target (not hard acceptance gate): file listing with pagination (default page size ≤ 100, sorted by `created_at DESC` or `name ASC`) should respond within 500ms p95 on a development-grade machine (e.g., 4-core, 16GB RAM, SSD), assuming appropriate database indexes on `(parent_folder_id, is_deleted, name)` and `(parent_folder_id, is_deleted, created_at)`. This target applies to warm-cache queries; cold-cache may be slower. Up to 10,000 items per folder. | P1 |
| NFR-PERF-03 | Share link generation shall complete in under 200ms. | P1 |
| NFR-PERF-04 | Database queries shall use appropriate indexes for all frequent access patterns. | P0 |

### 4.2 Security

| ID | Requirement | Priority |
|---|---|---|
| NFR-SEC-01 | All authentication and authorization must happen server-side. Never trust client-side claims. | P0 |
| NFR-SEC-02 | JWT tokens, passwords, share tokens, and MinIO credentials must never appear in logs. | P0 |
| NFR-SEC-03 | Passwords must be hashed with a strong algorithm (bcrypt cost >= 10 or equivalent). | P0 |
| NFR-SEC-04 | Prevent SQL injection (use parameterized queries or an ORM that does). | P0 |
| NFR-SEC-05 | Prevent path traversal on all user-supplied filename and path inputs. | P0 |
| NFR-SEC-06 | All API endpoints must require valid JWT authentication, with explicit exceptions: register (`POST /api/v1/auth/register`), login (`POST /api/v1/auth/login`), refresh (`POST /api/v1/auth/refresh`), and share link access (`GET /api/v1/shares/:token`). The refresh endpoint validates the refresh token (server-side state check per AUTH-04/AUTH-09), not an access JWT. The share link endpoint performs its own token validation (check expiry, revocation status, permission scope). No other public endpoints. | P0 |
| NFR-SEC-07 | File size limits shall be enforced before reading the request body. | P0 |
| NFR-SEC-08 | MIME types must be detected server-side; reject mismatches if validation is needed. | P0 |
| NFR-SEC-09 | Implement rate limiting on authentication endpoints to mitigate brute force attacks. | P1 |
| NFR-SEC-10 | CORS configuration must be restrictive in production. | P1 |
| NFR-SEC-11 | No timing or error-message side channels that leak information about private resources. | P1 |
| NFR-SEC-12 | Implement rate limiting on file upload and download endpoints (per-user and per-IP) to prevent storage bandwidth exhaustion. Limits must be configurable per deployment. | P1 |
| NFR-SEC-13 | File and folder rename/move operations must implement concurrency control (e.g., optimistic locking via a `version` column) to prevent lost updates from concurrent modifications. | P1 |
| NFR-SEC-14 | HTTP logging middleware must redact sensitive data from logged requests: route parameters (especially share tokens), raw URL paths, and query strings that may contain tokens or credentials. Standard request metadata (method, status code, duration, sanitized endpoint pattern) is acceptable. | P0 |

### 4.3 Scalability & Availability

| ID | Requirement | Priority |
|---|---|---|
| NFR-SCALE-01 | The system should handle concurrent uploads and downloads without significant degradation. | P1 |
| NFR-SCALE-02 | Static assets (frontend build) should be cacheable by CDN or browser. | P2 |
| NFR-SCALE-03 | Database connection pooling should be configured appropriately. | P1 |
| NFR-SCALE-04 | The system should support configurable file size limits per deployment. | P1 |

### 4.4 Reliability & Data Integrity

| ID | Requirement | Priority |
|---|---|---|
| NFR-REL-01 | MinIO writes and DB transactions are NOT atomic. Handle partial failure explicitly (e.g., upload succeeds but DB INSERT fails → clean up MinIO object). | P0 |
| NFR-REL-02 | File metadata must include SHA256 hash to detect corruption. | P0 |
| NFR-REL-03 | A configurable periodic cleanup mechanism shall permanently remove expired recycle bin items and must handle partial MinIO delete failures without leaving the DB in an inconsistent state. The cleanup interval must be configurable; the concrete scheduling mechanism (in-process goroutine, cron, etc.) is an implementation detail. | P1 |
| NFR-REL-04 | Permanent delete must handle MinIO cleanup failures gracefully (log and report, do not leave dangling DB records). | P0 |
| NFR-REL-05 | Rate-limited retry logic for MinIO operations where appropriate. | P2 |

### 4.5 Usability

| ID | Requirement | Priority |
|---|---|---|
| NFR-UI-01 | The web UI shall be responsive and work on desktop and tablet browsers. | P1 |
| NFR-UI-02 | Upload progress shall be visible to the user. | P1 |
| NFR-UI-03 | Error messages shall be user-friendly and not expose internal details. | P1 |
| NFR-UI-04 | The UI shall support drag-and-drop file upload. | P2 |

---

## 5. System Architecture

### 5.1 High-Level Architecture

```
┌─────────────────────────────────────────────────┐
│                    Browser                        │
│          (React + TypeScript SPA)                 │
└──────────────────────┬──────────────────────────┘
                        │ HTTP
                        ▼
┌─────────────────────────────────────────────────┐
│              Gin HTTP Server (Go)                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │ Auth MW  │ │ Handlers │ │ Middleware Chain  │  │
│  └──────────┘ └──────────┘ └──────────────────┘  │
│  ┌──────────────────────────────────────────┐    │
│  │           Service Layer                   │    │
│  └──────────────────────────────────────────┘    │
│  ┌──────────────────────────────────────────┐    │
│  │           Repository Layer                │    │
│  └──────────────────────────────────────────┘    │
└──────┬─────────────────────────────┬────────────┘
       │                             │
       ▼                             ▼
┌──────────────┐          ┌──────────────────┐
│  PostgreSQL  │          │  MinIO (S3)       │
│  (metadata)  │          │  (file objects)   │
└──────────────┘          └──────────────────┘
```

### 5.2 Backend Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── config/              # Configuration loading
│   ├── middleware/           # Auth, CORS, rate limiting, logging
│   ├── handler/             # HTTP handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   ├── model/               # Domain models / entities
│   ├── auth/                # JWT generation and validation
│   └── storage/             # MinIO abstraction layer
├── pkg/
│   └── ...                  # Shared utilities
├── migrations/              # Database migration files
├── go.mod
└── go.sum
```

> **Note**: This directory tree shows the **target structure** that will emerge as features are built. Do not pre-create all directories upfront — create them incrementally as each feature is implemented. See AGENTS.md.

### 5.3 Frontend Structure

```
frontend/
├── src/
│   ├── components/          # Reusable UI components
│   ├── pages/               # Route-level page components
│   ├── hooks/               # Custom React hooks
│   ├── services/            # API client / HTTP layer
│   ├── store/               # State management
│   ├── types/               # TypeScript type definitions
│   └── utils/               # Utility functions
├── public/
├── package.json
├── tsconfig.json
└── vite.config.ts
```

> **Note**: Same principle — create directories incrementally as components and pages are added.

### 5.4 API Contract

- API contracts are defined in `api/openapi.yaml` (OpenAPI 3.x).
- Contracts must be updated BEFORE writing or modifying handlers.
- Changes to request/response shape require the contract to be updated first.

---

## 6. API Design Principles

| Principle | Description |
|---|---|
| RESTful | Resource-oriented endpoints (`/api/v1/files`, `/api/v1/folders`, etc.) |
| Versioned | All endpoints prefixed with `/api/v1/` |
| JSON | Request and response bodies use JSON (file upload/download use multipart/stream) |
| Consistent Errors | Uniform error response schema: `{ "error": { "code": "...", "message": "..." } }` |
| Stateless (Hybrid) | Access tokens (short-lived, ~15min) are stateless JWT Bearer tokens in `Authorization` header. Refresh tokens (long-lived) require server-side state — stored in DB with rotation and revocation support. See JWT approach in §3.1. |
| Pagination | List endpoints support `?offset=&limit=` pagination |
| Cursor-based | (Future) Cursor-based pagination for large result sets |

### 6.1 Core Endpoints (Planned)

| Method | Path | Description |
|---|---|---|
| POST | /api/v1/auth/register | Register a new user |
| POST | /api/v1/auth/login | Log in, receive JWT |
| POST | /api/v1/auth/refresh | Refresh JWT token |
| POST | /api/v1/auth/logout | Log out |
| GET | /api/v1/files | List files in a folder |
| POST | /api/v1/files | Upload a file |
| GET | /api/v1/files/:id | Get file metadata |
| GET | /api/v1/files/:id/download | Download a file (stream) |
| PATCH | /api/v1/files/:id | Rename / move a file |
| DELETE | /api/v1/files/:id | Soft-delete a file |
| POST | /api/v1/folders | Create a folder |
| PATCH | /api/v1/folders/:id | Rename / move a folder |
| DELETE | /api/v1/folders/:id | Soft-delete a folder (cascade) |
| POST | /api/v1/shares | Create a share link |
| GET | /api/v1/shares/:token | Access a shared resource |
| DELETE | /api/v1/shares/:id | Revoke a share link |
| GET | /api/v1/recycle-bin | List recycle bin items |
| POST | /api/v1/recycle-bin/:id/restore | Restore from recycle bin |
| DELETE | /api/v1/recycle-bin/:id | Permanently delete |
| GET | /api/v1/user/me | Get current user profile |

---

## 7. Data Model Overview

### 7.1 Entities (PostgreSQL)

| Entity | Description | Key Fields |
|---|---|---|
| `users` | Registered users | id, email, password_hash, display_name, created_at, updated_at |
| `files` | File metadata | id, user_id, name, object_key, size, mime_type, sha256_hash, parent_folder_id, is_deleted, deleted_at, **version**, created_at, updated_at |
| `folders` | Folder metadata | id, user_id, name, parent_folder_id, is_deleted, deleted_at, **version**, created_at, updated_at |
| `share_links` | Share links | id, user_id, resource_type, resource_id, token, permission, expires_at, is_revoked, created_at |
| `refresh_tokens` | Refresh token store (enables rotation + server-side revocation) | id, user_id, token_hash, expires_at, is_revoked, created_at |
| `permissions` (Future) | ACL entries — **schema pending the folder permission inheritance decision (§8.3)** | AGENTS.md requires permissions in DB schema. Exact columns TBD until inheritance model is resolved — but the schema must be designed to accommodate ACL entries later (see note below) |
| `recycle_bin` | (Optional — can be tracked via `is_deleted` + `deleted_at` on files/folders) | — |

> **Note on `version`**: The `version` column (integer, monotonic) is used for optimistic locking on rename and move operations (see NFR-SEC-13). Clients must send the current `version` on update requests; the server rejects writes if the version has changed since read.
>
> **Note on owner semantics**: Throughout the schema, `user_id` in `files`, `folders`, and `share_links` tables represents the **resource owner** (creator/controller of the resource). This is the entity that AGENTS.md refers to as "owner". There is no separate `owner_id` field — `user_id` is the owning user.
>
> **Note on `permissions`**: AGENTS.md requires the DB schema to include permissions. For MVP, only two authorization models are implemented: (1) resource owner access (the `user_id` field on `files`/`folders`) and (2) share-link-based access (the `share_links` table). However, the schema must be designed to accommodate a permissions/ACL system later. The exact inheritance model (closure table, path enumeration, or adjacency-list with recursive CTEs) is still pending decision (§8.3) — do not finalize the `permissions`/`acls` table schema until resolved.

### 7.2 Storage Boundary Rules

- **PostgreSQL**: metadata only — users, files, folders, permissions, shares, recycle bin state.
- **MinIO**: raw file objects only.
- Do NOT store file contents in PostgreSQL.
- MinIO object keys are opaque UUIDs; user-supplied filenames are stored only in PostgreSQL metadata.
- Referential integrity: `user_id` and `parent_folder_id` are foreign keys.

---

## 8. Constraints

### 8.1 Technology Constraints

| Area | Constraint |
|---|---|
| Backend Language | **Go** — no exceptions |
| Backend Framework | **Gin** |
| Frontend | **React** + **TypeScript** (Vite) |
| Metadata Database | **PostgreSQL** |
| File Storage | **MinIO** (S3-compatible API) |
| Authentication | **JWT** |
| Architecture | **Backend monolith** — no microservices |
| API Definition | OpenAPI 3.x in `api/openapi.yaml` |

### 8.2 Development Constraints

| Constraint | Detail |
|---|---|
| Language | All code comments, docs, commit messages, and variable names must be in **English** |
| Dependency Policy | Technology stack choices (Go language, PostgreSQL, MinIO, JWT authentication, React + TypeScript + Vite) are fixed per §8.1 Technology Constraints. However, every individual Go module and npm package — including any that implement the tech stack (e.g., Gin, PostgreSQL driver, MinIO SDK, JWT library, bcrypt, testing frameworks, ESLint) — requires explicit approval per AGENTS.md Dependency Policy. Justify: necessity, alternatives considered, maintenance status, security impact. No dependency is "pre-approved". |
| Code Verification | Backend: `go vet ./...` + `go test ./...` + `gofmt -d .` (zero diffs) |
| Code Verification | Frontend: `npm run typecheck` + `npm run lint` + `npm run test` |
| File Safety | Uploads/downloads must stream; never read entire file into memory |
| Permission | Every file/folder operation must authorize — no exceptions |
| Mandatory Skill | Every implementation task (backend, frontend, API, config, docs) must load the `karpathy-guidelines` skill per AGENTS.md. Rules: think before coding, simplicity first, surgical changes, goal-driven execution. |

### 8.3 Pending Decisions (DO NOT Implement Without Resolution)

| Decision | Constraint |
|---|---|
| File preview approach | Do not add LibreOffice, ffmpeg, OCR, or any conversion services until resolved |
| Infrastructure (CI/CD, deployment) | Only basic local dev setup; no production infrastructure |
| Folder permission inheritance model | Must be explicitly designed before implementing folder-level permissions. **This decision directly impacts the database schema** — options include closure table, path enumeration, or adjacency-list with recursive CTEs. Do not finalize the `permissions`/`acls` table schema until resolved. See §7.1 note. |

### 8.4 Outright Bans

- `io.ReadAll` / `ioutil.ReadAll` — banned in production code paths.
- Bytea blobs in PostgreSQL — banned; file contents go to MinIO.
- `as any`, `@ts-ignore`, `@ts-expect-error` — banned in TypeScript.
- Exposing internal MinIO object keys or bucket URLs to the frontend — banned.
- Trusting client-supplied MIME types — banned.
- Logging credentials, tokens, or secrets — banned.

---

## 9. Out of Scope

The following features are explicitly **out of scope** for the initial release:

| Feature | Rationale |
|---|---|
| Server-side file conversion (LibreOffice, ffmpeg, OCR) | Pending decision; requires explicit approval |
| Full-text search | Complex; deferred |
| Real-time collaboration / WebSocket sync | Out of scope for MVP |
| Mobile native apps (iOS/Android) | Web-only for initial release |
| Multi-tenant / organization support | Single-user and small-team focus initially |
| S3-compatible external storage as primary backend | MinIO is the primary; external S3 is future |
| Encryption-at-rest (server-side) | Defer to infrastructure layer |
| Audit logging / compliance features | Future enhancement |
| Admin panel / user management dashboard | Future enhancement |
| Storage quota enforcement | Future enhancement |
| File versioning | Future enhancement |

---

## 10. Glossary

| Term | Definition |
|---|---|
| **JWT** | JSON Web Token — used for stateless authentication |
| **MinIO** | S3-compatible object storage server for storing raw file content |
| **S3** | Amazon Simple Storage Service — object storage protocol |
| **SHA256** | Secure Hash Algorithm 256-bit — used for file integrity verification |
| **Soft Delete** | Marking a record as deleted without removing it from the database, enabling restore |
| **Streaming** | Reading or writing data in chunks rather than loading the entire payload into memory |
| **Path Traversal** | A security attack where a user supplies `../` sequences to access files outside the intended directory |
| **TTL** | Time To Live — expiration duration for share links |
| **Opaque Key** | A storage key that reveals no information about the content or owner (e.g., UUID) |
| **OpenAPI** | A specification standard for describing RESTful APIs |
| **Gin** | A Go web framework used as the HTTP server |
