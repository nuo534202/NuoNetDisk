# NuoNetDisk - Detailed Design Document

> Version 1.0
> Status: Approved

---

## 1. Purpose

This detailed design continues the approved requirements, architecture design, overview design, and agent instructions for NuoNetDisk.

The goals of this document are:

- Define implementation-ready module behavior while staying within the approved monolithic layered architecture.
- Detail API inputs and outputs for the current MVP modules.
- Specify Handler, Service, Repository, and Storage responsibilities for every core workflow.
- Define authorization, database, MinIO, error, partial failure, concurrency, and test expectations.
- Preserve pending decisions without implementing them.

This document is intentionally written in English only and is suitable for later OpenAPI, backend, frontend, and test implementation work.

---

## 2. Source Alignment

This document is aligned with the following approved inputs:

- Requirements Document, version 1.0.
- Architecture Design Document, version 1.0.
- System Overview Design Document, version 1.0.
- Agent Instructions.

No approved document is modified by this detailed design.

Where a future API contract adjustment is mentioned, implementation must still follow the existing project rule: update `api/openapi.yaml` before writing or changing handlers.

---

## 3. Non-Goals

This phase does not include:

- Backend business source code.
- Frontend business source code.
- Complete migration SQL.
- New runtime dependencies.
- New development dependencies.
- Microservices.
- Server-side file preview conversion.
- Folder ACL inheritance implementation.
- File version history.
- Storage quota enforcement.
- Admin features.
- Audit logging.
- Production infrastructure or CI/CD.

---

## 4. Approved Architecture Constraints

The backend remains a monolith with layered boundaries:

```text
HTTP Handler -> Service -> Repository -> PostgreSQL
                          -> Storage -> MinIO
```

Layer rules:

| Layer | Allowed responsibilities | Forbidden responsibilities |
|---|---|---|
| Handler | HTTP parsing, validation, response formatting, stream handoff | Business logic, SQL, MinIO calls |
| Service | Business rules, authorization, orchestration, partial failure decisions | Gin-specific logic, direct HTTP response writes |
| Repository | PostgreSQL metadata access only | Business logic, MinIO calls, HTTP logic |
| Storage | MinIO object operations only | Metadata persistence, authorization decisions |
| Middleware | Cross-cutting HTTP behavior | Resource-specific business rules |

No file content is stored in PostgreSQL. MinIO stores raw file objects only.

---

## 5. Shared API Conventions

### 5.1 Base rules

| Item | Rule |
|---|---|
| Base path | `/api/v1` |
| Authentication | `Authorization: Bearer <access_token>` for protected endpoints |
| Public endpoints | Register, login, refresh, and share access only |
| Request bodies | JSON except upload and streaming responses |
| Upload | `multipart/form-data`, streamed |
| Download | Backend streaming response |
| List response | `{ "data": [...], "total": N }` |
| Error response | `{ "error": { "code": "...", "message": "..." } }` |
| Timestamp format | RFC 3339 |
| ID format | UUID |
| Pagination | `offset`, `limit`; max `limit` is 100 |
| Optimistic locking | Client sends current `version` in update requests |

### 5.2 Standard public DTOs

The following response shapes must never include MinIO object keys, bucket names, bucket URLs, credentials, password hashes, token hashes, or internal storage paths.

#### User DTO

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "display_name": "User",
  "created_at": "2026-01-01T00:00:00Z"
}
```

#### File DTO

```json
{
  "id": "uuid",
  "name": "report.pdf",
  "size": 12345,
  "mime_type": "application/pdf",
  "sha256_hash": "hex-encoded-sha256",
  "parent_folder_id": "uuid-or-null",
  "version": 1,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

#### Folder DTO

```json
{
  "id": "uuid",
  "name": "Documents",
  "parent_folder_id": "uuid-or-null",
  "version": 1,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

#### Share Link DTO

```json
{
  "id": "uuid",
  "resource_type": "file",
  "resource_id": "uuid",
  "permission": "preview",
  "expires_at": "2026-01-02T00:00:00Z",
  "is_revoked": false,
  "url": "https://frontend.example/shares/<opaque-token>",
  "created_at": "2026-01-01T00:00:00Z"
}
```

The `url` may include the share token because the owner needs it once to share the link. The token must not be logged.

---

## 6. Shared Security and Data Rules

### 6.1 Server-side authorization

Every file and folder operation must authorize in the Service layer. Middleware authentication is not enough.

The MVP authorization model is:

1. Owner access through `user_id` on files, folders, and share links.
2. Anonymous share-link access through a valid, unexpired, unrevoked share token with sufficient scope.

Unauthorized private resources should generally map to a not-found style response to avoid resource enumeration.

### 6.2 Sensitive data redaction

The system must never log:

- Passwords.
- Access tokens.
- Refresh tokens.
- Refresh token hashes.
- Share tokens.
- Authorization headers.
- MinIO credentials.
- MinIO object keys when avoidable.
- Raw request URLs containing sensitive path or query parameters.

Logs may contain request method, sanitized route pattern, status code, duration, request ID, user ID, and safe error category.

### 6.3 File safety rules

All file operations must follow these rules:

- Uploads stream through `io.Reader`.
- Downloads stream through `io.Writer`.
- Production code paths must not use `io.ReadAll`, `ioutil.ReadAll`, or equivalent full-body buffering.
- File size limits are checked before reading the request body.
- The request body is wrapped with a maximum-size reader before multipart parsing.
- The multipart file part is copied as a stream.
- MIME type is detected server-side from sniffed bytes.
- Client-supplied MIME type is treated as advisory only.
- User-supplied filenames are display metadata only.
- MinIO object keys are generated internally and are opaque.
- MinIO object keys and bucket URLs are never returned to clients.
- All storage access goes through backend services.

### 6.4 Opaque object key generation

Object keys must reveal no user identity, original filename, folder structure, or business meaning.

Approved shape:

```text
<random-or-uuid-token>
```

Optional internal prefixing for operational distribution is allowed only if it remains opaque, for example:

```text
objects/<opaque-random-token>
```

Do not use:

```text
<user-id>/<folder-name>/<filename>
```

### 6.5 MIME detection and streaming upload pipeline

The upload pipeline must support MIME sniffing without full buffering:

1. Handler validates `Content-Length` before body consumption.
2. Handler wraps the request body with an upload-size limiter.
3. Handler obtains a streaming multipart reader.
4. Handler locates exactly one file part.
5. Service reads up to 512 bytes from the beginning of the file stream for MIME detection.
6. Service reconstructs the stream by concatenating the sniffed bytes with the remaining reader.
7. Service wraps the reconstructed reader with hash computation.
8. Service passes the resulting stream directly to Storage upload.

Only the sniffed prefix is held in memory. The full file is never held in memory.

### 6.6 Hashing

SHA256 is computed during the upload stream. The hash is stored in PostgreSQL metadata after MinIO upload succeeds and before the metadata insert is committed.

The service should avoid a second read pass because request body streams may not be seekable.

### 6.7 Partial failure policy

PostgreSQL and MinIO are not atomic together. Every workflow that writes to both must define an explicit recovery strategy.

General rules:

| Situation | Required behavior |
|---|---|
| MinIO upload fails | Do not insert file metadata |
| MinIO upload succeeds, DB insert fails | Attempt MinIO cleanup; return error; log cleanup failure safely |
| DB soft-delete succeeds | Do not delete MinIO object |
| MinIO permanent delete fails | Keep related DB metadata in deleted state so the operation can be retried |
| MinIO delete succeeds, DB hard delete fails | Keep deleted metadata and allow retry; MinIO delete is idempotent |
| Folder permanent delete partially fails | Hard-delete only rows whose MinIO objects were successfully removed, or keep the full subtree if atomic user semantics are required by implementation; never restore an object known to be missing |
| Scheduled cleanup partially fails | Log safely, continue with other items, retry failed items on a later run |

### 6.8 Concurrency policy

| Workflow | Strategy |
|---|---|
| File rename | Optimistic locking with `version` |
| File move | Optimistic locking with `version` |
| Folder rename | Optimistic locking with `version` |
| Folder move | Optimistic locking with `version` |
| Refresh token rotation | Transactional revoke-and-create; detect replay |
| Soft-delete | Idempotent where practical |
| Restore | Transactional state change; validate parent state |
| Permanent delete | Idempotent storage delete; DB hard delete after storage success |
| Duplicate names | Repository enforces unique active sibling names through constraints or transactional checks |
| Folder move | Reject cycles using recursive ancestry validation |

### 6.9 Error mapping

| Domain condition | HTTP status | Code |
|---|---:|---|
| Invalid request | 400 | `INVALID_REQUEST` |
| Validation failure | 400 | `VALIDATION_ERROR` |
| Missing access token | 401 | `MISSING_TOKEN` |
| Invalid access token | 401 | `INVALID_TOKEN` |
| Refresh token replay | 401 | `TOKEN_REVOKED` |
| Forbidden operation | 403 or private-safe 404 | `FORBIDDEN` or `NOT_FOUND` |
| Resource not found | 404 | `NOT_FOUND` |
| Duplicate sibling name | 409 | `DUPLICATE` |
| Version mismatch | 409 | `CONFLICT` |
| File too large | 413 | `FILE_TOO_LARGE` |
| Rate limited | 429 | `RATE_LIMITED` |
| Unexpected error | 500 | `INTERNAL_ERROR` |

Share-link access failures must always return the same generic 404 body.

---

## 7. Data Model Detail

This section describes required fields and write behavior. It is not migration SQL.

### 7.1 `users`

Purpose: registered account identity.

Important fields:

- `id`
- `email`
- `password_hash`
- `display_name`
- `created_at`
- `updated_at`

Rules:

- `email` is normalized before storage and must be unique.
- `password_hash` is never returned.
- User deletion behavior is not part of MVP.

### 7.2 `refresh_tokens`

Purpose: server-side refresh token rotation and revocation state.

Important fields:

- `id`
- `user_id`
- `token_hash`
- `expires_at`
- `is_revoked`
- `created_at`

Rules:

- Client receives `token_id.secret`.
- DB stores only the hash of the secret.
- On refresh, the old row is marked revoked and a new row is created.
- Replay of a revoked token revokes all active refresh tokens for that user.

### 7.3 `folders`

Purpose: hierarchical folder metadata.

Important fields:

- `id`
- `user_id`
- `name`
- `parent_folder_id`
- `is_deleted`
- `deleted_at`
- `version`
- `created_at`
- `updated_at`

Rules:

- `user_id` is the owner.
- `parent_folder_id` is nullable for root-level folders.
- Only active sibling folders should have duplicate-name checks.
- Moving a folder must not create a cycle.
- Soft delete marks the subtree deleted in PostgreSQL only.

### 7.4 `files`

Purpose: file metadata only.

Important fields:

- `id`
- `user_id`
- `name`
- `object_key`
- `size`
- `mime_type`
- `sha256_hash`
- `parent_folder_id`
- `is_deleted`
- `deleted_at`
- `version`
- `created_at`
- `updated_at`

Rules:

- `object_key` is internal only.
- `object_key` is unique.
- `name` is display metadata and may be sanitized.
- `parent_folder_id` is nullable for root-level files.
- File content is stored only in MinIO.

### 7.5 `share_links`

Purpose: share-token access metadata.

Important fields:

- `id`
- `user_id`
- `resource_type`
- `resource_id`
- `token`
- `permission`
- `expires_at`
- `is_revoked`
- `created_at`

Rules:

- `user_id` is the owner who created the link.
- `resource_type` is `file` or `folder`.
- `permission` is `preview` or `download`.
- `expires_at` may be nullable only if the approved product behavior allows non-expiring links. MVP should prefer a configured maximum TTL.
- `token` is opaque and must be redacted from logs.
- A future hash-only token storage design may be considered separately, but this document does not introduce it.

---

## 8. Authentication and User Management

### 8.1 Functional goal

Provide registration, login, access-token issuance, refresh-token rotation, logout, and current-user profile retrieval.

### 8.2 Endpoints and DTOs

#### `POST /api/v1/auth/register`

Request:

```json
{
  "email": "user@example.com",
  "password": "PlaintextPassword123"
}
```

Response `201`:

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "display_name": "",
  "created_at": "2026-01-01T00:00:00Z"
}
```

#### `POST /api/v1/auth/login`

Request:

```json
{
  "email": "user@example.com",
  "password": "PlaintextPassword123"
}
```

Response `200`:

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<token-id>.<secret>",
  "expires_in": 900
}
```

#### `POST /api/v1/auth/refresh`

Request:

```json
{
  "refresh_token": "<token-id>.<secret>"
}
```

Response `200`:

```json
{
  "access_token": "<new-jwt>",
  "refresh_token": "<new-token-id>.<new-secret>",
  "expires_in": 900
}
```

#### `POST /api/v1/auth/logout`

Request:

```json
{
  "refresh_token": "<token-id>.<secret>"
}
```

Response:

```text
204 No Content
```

#### `GET /api/v1/user/me`

Response `200`:

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "display_name": "User",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### 8.3 Handler responsibilities

`AuthHandler`:

- Parse JSON body.
- Validate required fields exist.
- Reject malformed email syntax and empty password before calling service where possible.
- Never log request body.
- Call `AuthService`.
- Return token responses or public user DTOs.

`UserHandler`:

- Read authenticated user ID from middleware context.
- Call `UserService.Me`.
- Return public user DTO.

### 8.4 Service responsibilities

`AuthService`:

- Normalize email.
- Enforce password policy.
- Hash passwords.
- Verify password hashes.
- Generate access JWTs.
- Generate refresh tokens.
- Store refresh token hashes.
- Rotate refresh tokens.
- Detect replay of revoked refresh tokens.
- Revoke refresh tokens on logout.
- Return safe domain errors.

`UserService`:

- Load current user by ID.
- Hide sensitive fields.

### 8.5 Repository responsibilities

`UserRepository`:

- Create user.
- Get user by email.
- Get user by ID.
- Enforce email uniqueness.

`RefreshTokenRepository`:

- Get refresh token by ID.
- Create refresh token row.
- Mark token revoked.
- Revoke all active tokens for user.
- Run refresh rotation in a transaction.

### 8.6 Storage responsibilities

None. Auth does not use MinIO.

### 8.7 Core workflows

#### Registration

1. Handler parses request.
2. Service normalizes email.
3. Service validates password.
4. Repository checks email uniqueness.
5. Service hashes password.
6. Repository inserts user.
7. Handler returns public user DTO.

Failure handling:

- Duplicate email returns `409 DUPLICATE`.
- Invalid request returns `400`.
- Hashing or DB failures return safe `500`.

#### Login

1. Handler parses request.
2. Service loads user by normalized email.
3. Service verifies password.
4. Service generates access JWT.
5. Service generates refresh token and secret hash.
6. Repository inserts refresh token row.
7. Handler returns token pair.

Security notes:

- Invalid email and invalid password should return the same generic authentication error.
- Password and token values are never logged.
- Auth endpoints are rate-limited.

#### Refresh token rotation

1. Handler parses refresh token.
2. Service splits token into token ID and secret.
3. Repository loads token row by token ID.
4. Service verifies hash.
5. Service checks expiry and revocation state.
6. If revoked, service treats it as replay and revokes all active tokens for the user.
7. If valid, repository transaction revokes old token and inserts new token.
8. Service issues a new access token.
9. Handler returns the new token pair.

Concurrency strategy:

- Rotation must be transactionally serialized for the token row.
- Concurrent refresh requests with the same token should result in exactly one success.
- The losing request observes revoked state and triggers replay handling or returns `401`, depending on implementation policy.
- Replay handling must be careful not to revoke the newly created token from the winning request unless the project explicitly chooses strict replay semantics. The simpler and safer MVP behavior is: any second use of the old token revokes all active tokens for that user and forces re-login.

#### Logout

1. Handler requires authenticated user.
2. Handler parses refresh token.
3. Service verifies token belongs to current user if the token row can be identified.
4. Repository marks token revoked.
5. Handler returns `204`.

Logout is idempotent: logging out with an already revoked token returns `204` if it belongs to the user, or `401` if ownership cannot be verified safely.

### 8.8 Permission checks

- Register, login, and refresh do not require access JWTs.
- Logout requires access JWT plus refresh token verification.
- `GET /user/me` requires access JWT.
- Users can only read their own profile.

### 8.9 Database read/write points

| Operation | Reads | Writes |
|---|---|---|
| Register | `users` by email | Insert `users` |
| Login | `users` by email | Insert `refresh_tokens` |
| Refresh | `refresh_tokens`, `users` | Update old token, insert new token, possible revoke all |
| Logout | `refresh_tokens` | Update token revoked |
| Me | `users` by ID | None |

### 8.10 Exceptions and partial failures

- If access JWT generation fails after refresh token creation, revoke the newly created refresh token before returning an error.
- If refresh token creation fails after access token creation, do not return the access token.
- If logout token verification fails, return `401` with a safe message.
- DB transaction failure during refresh returns `500`; no partial token pair is returned.

### 8.11 Test scenarios

- Register succeeds with valid email and password.
- Register rejects duplicate email.
- Register stores password hash, not plaintext.
- Login succeeds and creates refresh token row.
- Login fails with wrong password and does not reveal whether email exists.
- Refresh succeeds and revokes the old token.
- Refresh rejects expired token.
- Refresh detects replay of revoked token.
- Concurrent refresh with the same token has at most one successful response.
- Logout revokes a valid refresh token.
- Logout rejects a token belonging to another user.
- `GET /user/me` requires JWT.
- No auth endpoint logs password or token values.

---

## 9. File Management

### 9.1 Functional goal

Provide secure owner-based file listing, metadata retrieval, streaming upload, streaming download, rename, move, and soft delete.

### 9.2 Endpoints and DTOs

#### `GET /api/v1/files`

Query parameters:

| Name | Required | Description |
|---|---|---|
| `folder_id` | No | Parent folder ID; omitted means root |
| `offset` | No | Pagination offset |
| `limit` | No | Pagination limit, max 100 |
| `sort` | No | `created_at` or `name` |
| `order` | No | `asc` or `desc` |

Response `200`:

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "report.pdf",
      "size": 12345,
      "mime_type": "application/pdf",
      "sha256_hash": "hex",
      "parent_folder_id": "uuid-or-null",
      "version": 1,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

#### `POST /api/v1/files`

Request:

```text
multipart/form-data
- file: binary file stream
- folder_id: optional UUID text field
```

Response `201`: File DTO.

#### `GET /api/v1/files/:id`

Response `200`: File DTO.

#### `GET /api/v1/files/:id/download`

Response `200`:

```text
Binary stream
```

Headers:

- `Content-Type`: stored MIME type.
- `Content-Length`: stored file size when known.
- `Content-Disposition`: safe attachment filename.
- `Accept-Ranges`: `bytes`.

#### `PATCH /api/v1/files/:id`

Request:

```json
{
  "name": "new-name.pdf",
  "parent_folder_id": "uuid-or-null",
  "version": 3
}
```

Response `200`: updated File DTO.

Fields are optional except `version`; at least one of `name` or `parent_folder_id` must be provided.

#### `DELETE /api/v1/files/:id`

Response:

```text
204 No Content
```

### 9.3 Handler responsibilities

`FileHandler`:

- Extract user ID from auth context.
- Parse path IDs and query parameters.
- Enforce request body size before reading upload body.
- Use a streaming multipart reader for upload.
- Do not call `ParseMultipartForm` in a way that buffers the full file.
- Pass the file stream to `FileService`.
- For download, set safe headers and stream the service-provided reader to the response writer.
- Close readers returned by services.
- Map domain errors to HTTP errors.

### 9.4 Service responsibilities

`FileService`:

- Authorize ownership for every file operation.
- Validate folder ownership for target parent folder.
- Sanitize display filename.
- Reject path traversal and empty names.
- Enforce file size based on metadata available before and during streaming.
- Detect MIME type server-side.
- Generate opaque object key.
- Compute SHA256 during upload.
- Coordinate MinIO upload and DB insert.
- Coordinate MinIO download and DB metadata read.
- Apply optimistic locking for rename and move.
- Soft-delete file metadata.
- Return public DTOs without object keys.

### 9.5 Repository responsibilities

`FileRepository`:

- List active files by owner and folder.
- Get active file by ID.
- Get deleted file by ID for recycle bin flows.
- Insert file metadata.
- Update name and parent folder with version check.
- Mark file deleted.
- Hard-delete file row after permanent deletion.
- Check active duplicate filename in target folder.
- Query object key for permanent deletion only.

`FolderRepository` is used by `FileService` to validate parent folder ownership and deletion state.

### 9.6 Storage responsibilities

`FileStorage`:

- Upload stream to MinIO.
- Download stream from MinIO.
- Delete object from MinIO.
- Treat delete of a missing object as success where supported.
- Never expose bucket URL or object key outside backend layers.
- Do not retry non-seekable upload streams at application level.

### 9.7 Core workflows

#### List files

1. Handler parses folder ID, pagination, sort, and order.
2. Service validates user ownership of folder if provided.
3. Repository queries active files for the owner and folder.
4. Handler returns list response.

Permission check:

- User must own the folder if `folder_id` is present.
- Root listing returns only files with matching owner and null parent.

DB points:

- Read folders for parent validation.
- Read files for listing and total count.

MinIO points:

- None.

#### Get file metadata

1. Handler parses file ID.
2. Service loads file by ID.
3. Service checks owner and active state.
4. Handler returns File DTO.

DB points:

- Read `files`.

MinIO points:

- None.

#### Streaming upload

1. Auth middleware validates JWT.
2. Handler checks `Content-Length` before reading request body.
3. Handler rejects missing or oversized `Content-Length` unless deployment policy explicitly allows chunked upload with a strict maximum reader wrapper.
4. Handler wraps the request body with the configured max body limit.
5. Handler creates a multipart reader and streams parts.
6. Handler passes the file part reader, original filename, declared part size if known, and parent folder ID to service.
7. Service validates parent folder ownership.
8. Service sanitizes display filename.
9. Service reads at most 512 bytes for MIME detection.
10. Service reconstructs the stream with the sniffed prefix.
11. Service wraps stream with SHA256 computation.
12. Service generates opaque object key.
13. Service uploads to MinIO using the stream and expected size.
14. Service inserts metadata into PostgreSQL.
15. If DB insert fails, service attempts MinIO cleanup.
16. Handler returns File DTO.

Important constraints:

- No full-file buffering.
- No client MIME trust.
- No object key exposure.
- MinIO upload happens before DB insert.
- Metadata insert happens only after MinIO success.

Partial failures:

| Failure | Behavior |
|---|---|
| Request too large before read | Return `413`, do not read body |
| Parent folder invalid | Return safe `404`, do not upload |
| MIME detection fails due to empty stream | Reject as validation error |
| MinIO upload fails | Return error, no DB insert |
| DB insert fails after upload | Attempt MinIO delete; log cleanup failure safely; return error |
| Cleanup delete fails | Log critical safe event; do not expose object key to client |

Concurrency:

- Duplicate active names in same folder are rejected.
- Duplicate check should be enforced transactionally or with a partial unique constraint if approved by schema design.
- Uploading the same display name concurrently should result in one success and one `409 DUPLICATE`, or both success only if duplicate filenames are explicitly allowed later.

#### Streaming download

1. Auth middleware validates JWT.
2. Handler parses file ID.
3. Service loads file metadata.
4. Service verifies owner and active state.
5. Service opens MinIO object stream using object key.
6. Handler sets headers from safe metadata.
7. Handler streams reader to response writer.
8. Handler closes the reader when complete or aborted.

Important constraints:

- No `io.ReadAll`.
- No direct MinIO URL.
- No object key exposure.
- Authorization happens before opening MinIO stream.

Partial failures:

| Failure | Behavior |
|---|---|
| File not found or not owned | Return safe `404` |
| File is soft-deleted | Return safe `404` |
| MinIO object missing | Return `500` or safe storage error before headers are committed |
| Client disconnects | Stop streaming, close reader, log safe debug event |
| Error after headers committed | Abort stream and log; response status may already be committed |

Range requests:

- Range support is required for future preview but preview is not implemented in MVP.
- If Range is implemented for download, validate range before opening the MinIO ranged reader.
- Return `206 Partial Content` only for valid ranges.
- Return `416` for invalid ranges if the API contract defines it.

#### Rename and move with optimistic locking

1. Handler parses file ID and JSON body.
2. Service requires `version`.
3. Service loads current file metadata.
4. Service checks owner and active state.
5. Service validates new name if present.
6. Service validates target folder if present.
7. Service checks duplicate active name in target folder.
8. Repository updates only if current DB version equals request version.
9. Repository increments version.
10. Handler returns updated File DTO.

Conflict behavior:

- If zero rows are updated because version changed, return `409 CONFLICT`.
- Client should re-fetch metadata and retry intentionally.

DB points:

- Read file.
- Read target folder.
- Read duplicate sibling.
- Write file metadata with version condition.

MinIO points:

- None. Rename and move affect metadata only.

#### Soft delete file

1. Handler parses file ID.
2. Service loads active file.
3. Service checks owner.
4. Repository marks `is_deleted=true` and sets `deleted_at`.
5. Handler returns `204`.

DB points:

- Read file.
- Update delete state.

MinIO points:

- None. Soft delete never removes object content.

### 9.8 Permission checks

- All file endpoints require JWT.
- User must own the file.
- User must own parent and target folders.
- Deleted files cannot be downloaded, renamed, moved, shared, or listed in normal file views.
- Private resource access failure should not reveal existence.

### 9.9 Test scenarios

- Upload streams without full-body read.
- Upload rejects missing file part.
- Upload rejects oversized body before reading.
- Upload detects MIME server-side.
- Upload stores SHA256.
- Upload creates opaque object key.
- Upload response does not include object key or bucket URL.
- Upload cleans MinIO object when DB insert fails.
- Download requires JWT.
- Download checks owner before opening MinIO stream.
- Download streams from MinIO to response writer.
- Download response does not expose object key.
- Rename rejects invalid names and path traversal.
- Rename returns `409` on stale version.
- Move rejects target folder owned by another user.
- Move rejects deleted target folder.
- Soft delete leaves MinIO object intact.
- Unauthorized file access returns safe response.
- Concurrent rename and move are protected by version checks.

---

## 10. Folder Management

### 10.1 Functional goal

Provide hierarchical folder creation, rename, move, and cascade soft delete for owner-controlled folders.

### 10.2 Endpoints and DTOs

#### `POST /api/v1/folders`

Request:

```json
{
  "name": "Documents",
  "parent_folder_id": "uuid-or-null"
}
```

Response `201`: Folder DTO.

#### `PATCH /api/v1/folders/:id`

Request:

```json
{
  "name": "Archive",
  "parent_folder_id": "uuid-or-null",
  "version": 2
}
```

Response `200`: updated Folder DTO.

Fields are optional except `version`; at least one of `name` or `parent_folder_id` must be provided.

#### `DELETE /api/v1/folders/:id`

Response:

```text
204 No Content
```

### 10.3 Handler responsibilities

`FolderHandler`:

- Parse user ID, path IDs, and JSON bodies.
- Validate required fields and UUID formats.
- Call `FolderService`.
- Return folder DTOs.
- Do not perform SQL or recursive traversal.

### 10.4 Service responsibilities

`FolderService`:

- Validate folder names.
- Reject path traversal and path-like names.
- Validate parent ownership and active state.
- Check duplicate active sibling folder names.
- Validate move does not create cycles.
- Apply optimistic locking for rename and move.
- Cascade soft delete through repository transaction.
- Return public folder DTOs.

### 10.5 Repository responsibilities

`FolderRepository`:

- Insert folder metadata.
- Get folder by ID.
- Get deleted folder by ID.
- List child folders if needed by folder views.
- Check duplicate active sibling names.
- Update folder name and parent with version condition.
- Compute descendant folder IDs using recursive query.
- Soft-delete folder subtree in a transaction.
- Restore folder subtree when called by recycle service.
- Hard-delete folder subtree when called by recycle service.

`FileRepository` is used for files contained in folder subtree during cascade delete, restore, and permanent delete flows.

### 10.6 Storage responsibilities

None for create, rename, move, and soft delete. MinIO is used only during recycle bin permanent delete of files inside folder subtrees.

### 10.7 Core workflows

#### Create folder

1. Handler parses request.
2. Service validates name.
3. Service validates parent folder if present.
4. Service checks duplicate active sibling folder.
5. Repository inserts folder metadata.
6. Handler returns Folder DTO.

DB points:

- Read parent folder.
- Read duplicate siblings.
- Insert folder.

MinIO points:

- None.

#### Rename folder

1. Handler parses folder ID and request body.
2. Service loads folder.
3. Service checks owner and active state.
4. Service validates new name.
5. Service checks duplicate active sibling name.
6. Repository updates with version condition and increments version.
7. Handler returns updated Folder DTO.

Concurrency:

- Stale version returns `409 CONFLICT`.
- Concurrent rename attempts result in one success.

#### Move folder

1. Handler parses folder ID and target parent folder ID.
2. Service loads source folder.
3. Service checks owner and active state.
4. Service validates target parent is owned by user and active.
5. Service rejects moving folder under itself or any descendant.
6. Service checks duplicate active sibling name in target parent.
7. Repository updates parent with version condition and increments version.
8. Handler returns updated Folder DTO.

Cycle prevention:

- Repository or service must query ancestors or descendants.
- If target parent is in source subtree, reject with `400 VALIDATION_ERROR`.

#### Folder cascade soft delete

1. Handler parses folder ID.
2. Service loads folder and checks owner.
3. Repository computes descendant folder IDs.
4. Repository starts transaction.
5. Repository marks all files in the subtree deleted.
6. Repository marks all folders in the subtree deleted.
7. Repository commits transaction.
8. Handler returns `204`.

Important constraints:

- No MinIO objects are removed.
- Soft delete is DB-only.
- All subtree rows receive delete state.
- Operation is idempotent if repeated on already deleted folders, returning either `204` or safe `404` according to service policy.

### 10.8 Permission checks

- All folder endpoints require JWT.
- User must own source folder.
- User must own target parent folder.
- Deleted folders cannot be renamed, moved, or used as upload targets.
- Moving across users is forbidden.
- Folder ACL inheritance is not implemented.

### 10.9 Partial failure handling

- Create and update are DB-only and should be transactional where multiple checks and writes must be consistent.
- Cascade soft delete uses a DB transaction. If any update fails, rollback.
- Since no MinIO operation occurs during soft delete, there is no DB-MinIO partial failure in this workflow.

### 10.10 Test scenarios

- Create folder in root.
- Create folder under valid parent.
- Create rejects invalid name and path traversal.
- Create rejects parent owned by another user.
- Create rejects duplicate active sibling name.
- Rename succeeds with correct version.
- Rename returns `409` with stale version.
- Move succeeds to root.
- Move succeeds to another folder owned by same user.
- Move rejects cycle.
- Move rejects deleted target parent.
- Cascade soft delete marks folder subtree and contained files deleted.
- Cascade soft delete rolls back on repository failure.
- Folder operations never call MinIO except through recycle bin permanent delete.

---

## 11. Recycle Bin

### 11.1 Functional goal

Provide listing, restore, and permanent deletion of soft-deleted files and folders while preserving data integrity across PostgreSQL and MinIO.

### 11.2 Endpoints and DTOs

#### `GET /api/v1/recycle-bin`

Query parameters:

| Name | Required | Description |
|---|---|---|
| `offset` | No | Pagination offset |
| `limit` | No | Pagination limit, max 100 |

Response `200`:

```json
{
  "data": [
    {
      "id": "uuid",
      "type": "file",
      "name": "report.pdf",
      "parent_folder_id": "uuid-or-null",
      "deleted_at": "2026-01-01T00:00:00Z",
      "size": 12345,
      "mime_type": "application/pdf",
      "version": 3
    }
  ],
  "total": 1
}
```

#### `POST /api/v1/recycle-bin/:id/restore`

Request:

```json
{
  "type": "file"
}
```

Response `200`: restored File DTO or Folder DTO.

The current architecture endpoint includes only `:id`. Because file IDs and folder IDs may both be UUIDs, the detailed request includes `type` to avoid ambiguous lookup. The OpenAPI contract must define this before implementation.

#### `DELETE /api/v1/recycle-bin/:id`

Request:

```json
{
  "type": "file",
  "confirm": true
}
```

Response:

```text
204 No Content
```

The explicit confirmation requirement can be implemented as a JSON body, a confirmation header, or a UI-only confirmation step. The API contract must choose one before handler implementation. The service must not permanently delete when confirmation is absent.

### 11.3 Handler responsibilities

`RecycleHandler`:

- Require JWT.
- Parse pagination.
- Parse item ID and type.
- Enforce explicit confirmation for permanent delete.
- Call `RecycleService`.
- Return restored metadata or `204`.

### 11.4 Service responsibilities

`RecycleService`:

- List deleted files and folders owned by user.
- Restore deleted file.
- Restore deleted folder subtree.
- Resolve missing or deleted parent folders.
- Permanently delete file with MinIO cleanup.
- Permanently delete folder subtree with MinIO cleanup.
- Coordinate partial failure handling.
- Preserve safe responses for unauthorized resources.

### 11.5 Repository responsibilities

`FileRepository`:

- List deleted files by owner.
- Get deleted file.
- Restore file.
- Hard-delete file rows.
- Get object keys for permanent delete.

`FolderRepository`:

- List deleted folders by owner.
- Get deleted folder.
- Compute folder subtree.
- Restore folder subtree.
- Hard-delete folder rows.
- Query files in deleted folder subtree.

### 11.6 Storage responsibilities

`FileStorage`:

- Delete MinIO objects during permanent delete.
- Treat missing object as success where possible.
- Return errors without secrets.
- Support repeated delete attempts.

### 11.7 Core workflows

#### List recycle bin

1. Handler parses pagination.
2. Service queries deleted files and folders owned by user.
3. Repository returns paginated combined items or service merges separate paginated lists according to implementation plan.
4. Handler returns recycle item DTOs.

DB points:

- Read `files` where deleted and owned.
- Read `folders` where deleted and owned.

MinIO points:

- None.

#### Restore file

1. Handler parses item ID and `type=file`.
2. Service loads deleted file by ID.
3. Service checks owner.
4. Service resolves original parent folder:
   - If parent is active, restore into original parent.
   - If parent is deleted and still exists, restore parent chain first or restore file to root according to product policy.
   - If parent was permanently deleted, restore to root.
5. Service checks duplicate active name in restore target.
6. Repository clears delete state and increments version if needed.
7. Handler returns File DTO.

Conflict policy:

- If duplicate active name exists in restore target, return `409 DUPLICATE`.
- The frontend can ask the user to rename or move later.
- Automatic renaming is not part of MVP unless OpenAPI defines it.

MinIO points:

- None. The object should still exist because soft delete does not remove it.
- If object is missing during a later download, that is a storage integrity error.

#### Restore folder

1. Handler parses item ID and `type=folder`.
2. Service loads deleted folder by ID.
3. Service checks owner.
4. Service resolves original parent folder.
5. Service checks duplicate active name in restore target.
6. Repository restores folder subtree in a transaction.
7. Repository clears delete state for contained files and subfolders that are still present.
8. Handler returns Folder DTO.

Restore preservation rule:

- Preserve original directory structure as much as possible.
- If an ancestor is still deleted and restorable, restore ancestor chain.
- If an ancestor was permanently deleted, restore the selected folder to root.
- Do not recreate permanently deleted records.

#### Permanent delete file

1. Handler requires explicit confirmation.
2. Service loads deleted file by ID.
3. Service checks owner.
4. Service calls Storage delete with object key.
5. If storage delete succeeds, repository hard-deletes file row.
6. If repository hard-delete fails, metadata remains deleted and deletion can be retried.
7. Handler returns `204` only when the DB hard delete succeeds.

Failure behavior:

| Failure | Behavior |
|---|---|
| File is not deleted | Return safe `404` or validation error according to API contract |
| Storage delete fails | Keep DB row deleted, return safe error, allow retry |
| Storage delete succeeds but DB delete fails | Keep DB row deleted, return `500`, allow retry; later storage delete is idempotent |
| File object already missing | Treat storage delete as success if the storage layer can confirm missing-object semantics |

#### Permanent delete folder

1. Handler requires explicit confirmation.
2. Service loads deleted folder by ID.
3. Service checks owner.
4. Repository computes deleted folder subtree and contained files.
5. Service attempts MinIO delete for each contained file object.
6. Service records per-object success or failure in memory for this request.
7. If any storage delete fails:
   - Keep failed file rows in deleted state.
   - Do not restore them.
   - Prefer not to hard-delete parent folder rows that still contain failed files.
   - Return a safe error after processing all attempted objects.
8. For successfully deleted objects, repository may hard-delete corresponding file rows in a transaction.
9. Repository hard-deletes folders only when no remaining child records require the folder for retry or restore.
10. Handler returns `204` only when all object deletes and DB hard deletes succeed.

Rationale:

- Deleting DB rows before MinIO success can orphan raw objects.
- Keeping failed rows in deleted state allows retry.
- MinIO delete is idempotent, so retry is safe after partial success.
- The user-visible operation may fail after partial progress, but storage and metadata remain recoverable or retryable.

#### Scheduled cleanup

1. Cleanup runner identifies expired deleted files and folders.
2. It processes items in bounded batches.
3. It calls the same service-level permanent delete logic or a shared internal deletion coordinator.
4. It logs safe errors and continues with subsequent items.
5. Failed items remain deleted and are retried in a later cycle.

No queue or new scheduler dependency is introduced. The approved in-process periodic cleanup remains sufficient for MVP.

### 11.8 Permission checks

- All recycle bin endpoints require JWT.
- User can only list, restore, or permanently delete own deleted resources.
- Share links do not grant recycle bin access.

### 11.9 Concurrency strategy

- Restore and permanent delete racing on the same item must be serialized by repository state checks.
- Permanent delete should only operate on rows where `is_deleted=true`.
- Restore should only operate on rows where `is_deleted=true`.
- A state change that affects zero rows means another request won and should return `409` or safe `404`.
- Scheduled cleanup and user-initiated permanent delete use the same idempotent rules.

### 11.10 Test scenarios

- List recycle bin shows only current user's deleted items.
- Restore file to original active parent.
- Restore file to root when parent was permanently removed.
- Restore rejects duplicate active name.
- Restore folder preserves subtree.
- Restore folder handles deleted ancestor chain.
- Permanent delete file deletes MinIO object before DB row.
- Permanent delete file keeps DB row if MinIO delete fails.
- Permanent delete file can be retried after DB hard-delete failure.
- Permanent delete folder handles partial MinIO failures without orphaning objects.
- Scheduled cleanup continues after one item fails.
- Recycle bin operations reject cross-user access.
- Permanent delete requires explicit confirmation.

---

## 12. Share Links

### 12.1 Functional goal

Allow owners to create expirable and revocable share links for files or folders and allow anonymous access through share-token validation without exposing internal storage details.

### 12.2 Endpoints and DTOs

#### `POST /api/v1/shares`

Request:

```json
{
  "resource_type": "file",
  "resource_id": "uuid",
  "permission": "download",
  "expires_in_hours": 24
}
```

Response `201`:

```json
{
  "id": "uuid",
  "resource_type": "file",
  "resource_id": "uuid",
  "permission": "download",
  "expires_at": "2026-01-02T00:00:00Z",
  "is_revoked": false,
  "url": "https://frontend.example/shares/<opaque-token>",
  "created_at": "2026-01-01T00:00:00Z"
}
```

#### `DELETE /api/v1/shares/:id`

Response:

```text
204 No Content
```

#### `GET /api/v1/shares/:token`

Default JSON response for file resource:

```json
{
  "resource_type": "file",
  "permission": "download",
  "file": {
    "id": "uuid",
    "name": "report.pdf",
    "size": 12345,
    "mime_type": "application/pdf",
    "sha256_hash": "hex",
    "version": 1,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
}
```

Default JSON response for folder resource:

```json
{
  "resource_type": "folder",
  "permission": "preview",
  "folder": {
    "id": "uuid",
    "name": "Shared Folder",
    "version": 1
  },
  "children": {
    "folders": [],
    "files": []
  }
}
```

Optional file download mode:

```text
GET /api/v1/shares/:token?download=1
```

Response when the token targets a file and permission is `download`:

```text
Binary stream
```

This preserves the approved public route shape while providing explicit download access. The OpenAPI contract must define this query mode before implementation.

### 12.3 Handler responsibilities

`ShareHandler`:

- For create and revoke, require JWT and read owner user ID.
- For anonymous access, do not require JWT.
- Redact share token from logs.
- Parse resource type, resource ID, permission, and expiry.
- For anonymous download mode, stream the service-provided reader without buffering.
- Return the same generic 404 for every share access validation failure.

### 12.4 Service responsibilities

`ShareService`:

- Verify owner can share the resource.
- Reject deleted or inaccessible resources.
- Validate permission.
- Validate expiry against configured maximum TTL.
- Generate opaque share token.
- Store share metadata.
- Revoke links by owner.
- Validate anonymous share token.
- Check expiry and revocation.
- Load shared file metadata or folder listing.
- Stream shared file content only when permission allows download mode.
- Return generic not-found errors for all anonymous share failures.

### 12.5 Repository responsibilities

`ShareRepository`:

- Insert share link.
- Get share link by token.
- Get share link by ID.
- Revoke share link by ID and owner.
- Optionally list share links by owner if a management UI is later defined.

`FileRepository` and `FolderRepository`:

- Load target resource.
- Validate owner for creation.
- Validate active state for access.
- List folder children for shared folder access.

### 12.6 Storage responsibilities

`FileStorage`:

- Download object stream for anonymous file download only after share validation.
- No storage operation for share creation or revocation.

### 12.7 Core workflows

#### Create share link

1. Auth middleware validates JWT.
2. Handler parses request.
3. Service validates resource type and permission.
4. Service loads target file or folder.
5. Service verifies owner access.
6. Service rejects deleted resource.
7. Service validates expiry against configuration.
8. Service generates opaque token.
9. Repository inserts share link.
10. Handler returns metadata and share URL.

DB points:

- Read target resource.
- Insert `share_links`.

MinIO points:

- None.

Partial failures:

- If token generation fails, no DB write occurs.
- If DB insert fails, no share URL is returned.

Concurrency:

- Multiple active links for the same resource may be allowed unless product requirements later restrict them.
- Token uniqueness is enforced by repository constraints; on collision, retry token generation a bounded number of times.

#### Revoke share link

1. Auth middleware validates JWT.
2. Handler parses share ID.
3. Service loads share link or updates by owner.
4. Repository marks `is_revoked=true`.
5. Handler returns `204`.

Rules:

- Only the owner can revoke.
- Revocation is idempotent.
- Anonymous access with revoked token returns generic `404`.

DB points:

- Read or update `share_links`.
- No MinIO operation.

#### Expiry handling

Expiry is enforced at access time:

1. Repository loads share link by token.
2. Service compares `expires_at` with current server time.
3. Expired links are treated like not found.
4. Optional cleanup of expired links is a future maintenance optimization, not required for correctness.

#### Anonymous metadata access

1. Public route bypasses JWT middleware.
2. Handler passes token to service without logging it.
3. Service loads share link by token.
4. Service validates not expired, not revoked, and supported permission.
5. Service loads target resource.
6. Service rejects deleted or inaccessible resource.
7. Service returns safe file metadata or folder listing.
8. Handler returns JSON.

All failures return:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Share link not found or expired"
  }
}
```

#### Anonymous download access

1. Public route bypasses JWT middleware.
2. Handler detects `download=1`.
3. Service validates token exactly as metadata access does.
4. Service verifies resource type is `file`.
5. Service verifies permission is `download`.
6. Service opens MinIO stream.
7. Handler sets safe download headers.
8. Handler streams content to the anonymous client.

Important constraints:

- `preview` permission must not allow download mode.
- The stream goes through backend.
- The response does not include object key or bucket URL.
- All token failures and permission mismatches return the same generic `404`.

### 12.8 Permission checks

- Create and revoke require JWT and owner access.
- Anonymous access requires valid share token.
- Share token grants only the configured scope.
- Share token never grants recycle bin, rename, move, delete, or owner operations.
- Folder share access is read-only.

### 12.9 Partial failure handling

- Share creation is DB-only after token generation.
- Share revocation is DB-only and idempotent.
- Anonymous download has no DB writes and one MinIO read.
- If MinIO download fails before headers are committed, return safe storage error.
- If MinIO download fails after streaming starts, log safely and terminate stream.

### 12.10 Concurrency strategy

- Revoke racing with access:
  - If access validates before revocation commits, one request may complete.
  - New access after revocation commits must fail.
- Expiry racing with access:
  - Service uses a single server-time check per request.
- Token collision:
  - Repository unique constraint catches collision.
  - Service retries bounded token generation.

### 12.11 Test scenarios

- Create share link for owned file.
- Create share link for owned folder.
- Create rejects resource owned by another user.
- Create rejects deleted resource.
- Create validates TTL.
- Revoke succeeds for owner.
- Revoke rejects other user.
- Revoked link returns generic 404.
- Expired link returns generic 404.
- Missing token returns same generic 404 shape.
- Permission mismatch returns same generic 404 shape.
- Anonymous metadata access returns safe DTOs only.
- Anonymous download streams only when permission is `download`.
- Anonymous access never exposes MinIO object key or bucket URL.
- Share token is redacted from logs.

---

## 13. Frontend Integration Design

### 13.1 Functional goal

Provide React + TypeScript UI integration with the backend API using the approved frontend stack and no additional state-management dependency.

### 13.2 Frontend modules

| Module | Responsibility |
|---|---|
| Auth pages | Register, login, logout |
| Auth state | Store access token in memory, track current user |
| API client | Typed fetch wrapper with error handling |
| File browser | List folders and files, navigate folders |
| Upload UI | Stream browser file upload through `FormData` |
| Download UI | Trigger backend download without exposing MinIO URL |
| Rename and move UI | Send current version for optimistic locking |
| Recycle bin UI | List, restore, permanently delete with confirmation |
| Share UI | Create, copy, and revoke share links |
| Anonymous share page | Access shared resource via token route |

### 13.3 Frontend constraints

- Do not store access token in localStorage.
- Do not log tokens.
- Do not expose object keys.
- Use native `fetch` by default.
- Do not introduce axios or additional state libraries without approval.
- Do not implement preview until the pending preview decision is resolved.

### 13.4 Frontend test scenarios

- Login stores access token only in in-memory state.
- API client attaches bearer token to protected requests.
- API client does not attach JWT to anonymous share access.
- Upload sends multipart form data.
- Download uses backend URL only.
- Rename and move send `version`.
- `409 CONFLICT` prompts refresh or retry UI.
- Permanent delete requires explicit confirmation.
- Share page handles generic 404 without exposing reason.

---

## 14. Module Interaction Summary

### 14.1 Upload

```text
FileHandler -> FileService -> FolderRepository
                           -> FileStorage.Upload
                           -> FileRepository.Insert
```

### 14.2 Download

```text
FileHandler -> FileService -> FileRepository.Get
                           -> FileStorage.Download
```

### 14.3 Rename or move

```text
FileHandler -> FileService -> FileRepository.Get
                           -> FolderRepository.Get
                           -> FileRepository.UpdateWithVersion
```

### 14.4 Folder cascade soft delete

```text
FolderHandler -> FolderService -> FolderRepository.SoftDeleteSubtreeTx
                              -> FileRepository.SoftDeleteByFolderSubtreeTx
```

### 14.5 Recycle restore

```text
RecycleHandler -> RecycleService -> FileRepository or FolderRepository
                                -> parent resolution repositories
                                -> restore transaction
```

### 14.6 Permanent delete

```text
RecycleHandler -> RecycleService -> FileRepository or FolderRepository
                                -> FileStorage.Delete
                                -> hard-delete transaction
```

### 14.7 Share access

```text
ShareHandler -> ShareService -> ShareRepository.GetByToken
                            -> FileRepository or FolderRepository
                            -> FileStorage.Download when download mode is allowed
```

---

## 15. Repository Method Expectations

The exact Go signatures are implementation details, but repositories should provide the following capabilities through service-owned interfaces.

### 15.1 User repository

- Create user.
- Get user by email.
- Get user by ID.

### 15.2 Refresh token repository

- Create token.
- Get token by ID.
- Revoke token by ID.
- Revoke all active tokens for user.
- Rotate token in transaction.

### 15.3 File repository

- Insert file.
- Get active file by ID.
- Get deleted file by ID.
- List active files by folder.
- List deleted files by owner.
- Update file with version condition.
- Soft-delete file.
- Restore file.
- Hard-delete file.
- List files in folder subtree.
- Check duplicate active file name.

### 15.4 Folder repository

- Insert folder.
- Get active folder by ID.
- Get deleted folder by ID.
- List active folders by parent.
- List deleted folders by owner.
- Update folder with version condition.
- Validate ancestry or descendants.
- Soft-delete subtree in transaction.
- Restore subtree in transaction.
- Hard-delete subtree when safe.
- Check duplicate active folder name.

### 15.5 Share repository

- Insert share link.
- Get share link by token.
- Get share link by ID and owner.
- Revoke share link.
- Optionally list share links by owner.

---

## 16. Storage Method Expectations

The storage interface should support:

- Upload object from stream.
- Download object as stream.
- Download object range if range support is implemented.
- Delete object idempotently.
- Copy object only if a future design requires it.

Storage methods must:

- Accept context.
- Return errors without secrets.
- Avoid full-file buffering.
- Avoid application-level retries for non-seekable upload streams.
- Hide MinIO bucket and object details from handlers and clients.

---

## 17. Testing Strategy

### 17.1 Unit tests

Unit tests should cover services with fake repositories and fake storage:

- Auth rotation and replay.
- File upload partial failure.
- File download authorization.
- File rename and move conflict.
- Folder cycle detection.
- Folder cascade soft delete.
- Recycle restore parent resolution.
- Permanent delete storage failure.
- Share token validation.
- Anonymous share access generic 404 behavior.

### 17.2 Handler tests

Handler tests should cover:

- Request validation.
- Auth middleware behavior.
- Upload size rejection before body read.
- Streaming upload path with a controlled reader.
- Download response headers.
- Error response shape.
- Share token route redaction behavior where testable.

### 17.3 Repository tests

Repository tests should cover with PostgreSQL test setup when available:

- Unique email.
- Active duplicate sibling checks.
- Optimistic locking updates.
- Recursive folder subtree queries.
- Soft-delete transaction rollback.
- Restore transaction behavior.
- Hard-delete ordering.

### 17.4 Storage tests

Storage tests should cover with fake storage first and MinIO integration when environment is available:

- Upload called with expected object key.
- Download returns streaming reader.
- Delete is idempotent.
- Delete failure propagates to service.
- No test depends on object key being user-readable.

### 17.5 Security tests

Security tests should cover:

- Cross-user file access.
- Cross-user folder access.
- Cross-user share revocation.
- Share token failure equivalence.
- Token and secret redaction.
- Path traversal rejection.
- Object key non-exposure.
- MIME detection independent from client header.

---

## 18. Implementation Guardrails

Before implementation:

1. Update `api/openapi.yaml` for any request or response shape used here that is not already in the API contract.
2. Request explicit approval for any new Go or npm dependency.
3. Implement the smallest vertical slice first.
4. Keep directories incremental.
5. Add tests for file and permission changes.
6. Run the project verification commands that exist at that time.
7. Do not claim unavailable commands were run.

This document itself introduces no new runtime dependency, development dependency, microservice, or business source code.
