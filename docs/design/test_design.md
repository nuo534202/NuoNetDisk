# NuoNetDisk - Test Design Document

> Version 1.0
> Status: Approved

---

## 1. Test Design Principles

1. Keep tests aligned with the approved backend monolith and layered architecture.
2. Test Handler, Service, Repository, and Storage behavior at the proper boundary.
3. Prioritize security, authorization, streaming safety, partial failure handling, and regression safety.
4. Do not introduce any new runtime dependency, development dependency, or test dependency in this phase.
5. Do not design implementation tests for unresolved or out-of-scope features.
6. Keep all test names, fixture names, messages, and documentation in English only.
7. Avoid relying on MinIO object keys, bucket names, password hashes, token hashes, or internal paths in client-visible assertions.
8. Prefer deterministic fixtures and fakes for unit tests; use PostgreSQL and MinIO integration tests only through the approved local development environment when it exists.

---

## 2. Test Scope

### 2.1 Unit Tests

Unit tests cover isolated business logic with fakes or stubs:

- Authentication service logic.
- Refresh token rotation, replay detection, and logout revocation.
- File upload orchestration, MIME detection, hashing, object key generation, and partial failure cleanup.
- File download authorization and stream handoff.
- File rename and move optimistic locking decisions.
- Folder validation, move cycle detection, and cascade soft-delete orchestration.
- Recycle bin restore and permanent delete decisions.
- Share token validation, expiry, revocation, permission scope, and generic 404 mapping.
- Error mapping and sensitive data redaction helpers.

### 2.2 Integration Tests

Integration tests cover real boundaries when the approved local environment is available:

- Repository behavior against PostgreSQL.
- Storage behavior against MinIO.
- Service workflows that coordinate PostgreSQL metadata and MinIO objects.
- Transaction behavior, rollback, idempotency, and retry-safe states.
- Scheduled recycle-bin cleanup behavior when the scheduling mechanism is available.

No new database test framework, migration runner, container test framework, or MinIO test dependency is introduced by this design.

### 2.3 API Tests

API tests cover HTTP request and response behavior through Gin handlers and middleware:

- Public and protected endpoint access.
- Request validation and error response shape.
- Upload size rejection before body consumption.
- Multipart upload behavior.
- Streaming download behavior and response headers.
- Share access without JWT.
- Generic 404 behavior for share-link failures.
- Route logging redaction where observable.

### 2.4 Frontend Tests

Frontend tests cover React + TypeScript UI behavior at component, hook, and API client boundaries:

- Login, logout, and session state.
- API client request construction.
- Folder browsing and file listing states.
- Upload form, drag-and-drop entry point, and upload progress state.
- Rename, move, delete, restore, permanent delete, and share flows.
- User-facing error handling without internal details.
- Anonymous share page behavior.

No new frontend test dependency is introduced by this design.

### 2.5 Security Tests

Security tests cover negative and abuse-oriented behavior:

- JWT validation.
- Missing, invalid, expired, and malformed tokens.
- Refresh token replay detection.
- Cross-user access denial.
- Private resource non-enumeration.
- Share token failure equivalence.
- Path traversal rejection.
- Server-side MIME detection.
- Sensitive data redaction in logs.
- Non-exposure of MinIO object keys, bucket URLs, credentials, password hashes, refresh token hashes, and share tokens.

### 2.6 Regression Tests

Regression tests are the stable P0/P1 subset that must run before merging changes to file, folder, auth, share, recycle bin, middleware, or frontend API behavior:

- Auth smoke tests.
- Upload and download streaming tests.
- Authorization tests for every file and folder operation.
- Recycle bin soft-delete, restore, and permanent delete tests.
- Share creation, access, expiry, revocation, and generic 404 tests.
- Redaction tests.
- Frontend login, upload, conflict, delete, restore, and share UI flow tests.

---

## 3. Test Strategy

### 3.1 Backend Handler, Service, Repository, and Storage Layer Strategy

| Layer | Main strategy | Key assertions |
|---|---|---|
| Handler | Use HTTP-level tests with fake services where possible. | Correct request parsing, validation, status codes, response shape, streaming handoff, and no business logic in handlers. |
| Service | Use fake repositories and fake storage to isolate business rules. | Authorization is performed, partial failures are handled, object keys are generated internally, refresh tokens rotate, and domain errors are safe. |
| Repository | Use PostgreSQL integration tests when an approved local DB is available. | Unique constraints, sibling-name checks, optimistic locking, recursive folder queries, soft-delete transactions, and hard-delete ordering. |
| Storage | Use fake storage for service unit tests and MinIO integration tests when available. | Upload and download stream behavior, opaque object keys, idempotent delete, and failure propagation. |
| Middleware | Use Gin route tests with controlled requests. | JWT validation, public endpoint whitelist, redacted logging, panic recovery behavior, CORS and rate-limit behavior where implemented. |

### 3.2 Backend Layer-Specific Design Rules

- Handlers must not call repositories or storage directly.
- Services must not depend on `gin.Context` or write HTTP responses.
- Repositories must not call MinIO or perform business authorization.
- Storage must not persist metadata or make authorization decisions.
- File content must not be stored in PostgreSQL.
- Upload and download paths must avoid full-body buffering.

### 3.3 Frontend Component, Hook, and API Client Strategy

| Area | Strategy | Key assertions |
|---|---|---|
| Components | Render core states and user actions with mocked API calls. | Correct forms, buttons, confirmation dialogs, disabled states, error messages, and loading states. |
| Hooks | Test state transitions around authentication, listing, upload, and sharing. | Token state, optimistic UI refresh, conflict handling, and retry prompts. |
| API client | Test request construction without real network calls. | Correct method, path, headers, JSON body, multipart body, bearer token behavior, and anonymous share access behavior. |
| UI flows | Use integration-style frontend tests where the approved frontend test setup exists. | Login, folder browsing, upload, rename, move, delete, recycle bin restore, permanent delete, share creation, and anonymous share page behavior. |

Frontend tests must not assume direct access to MinIO or internal object keys. The frontend must use backend APIs only.

### 3.4 Database and MinIO Integration Strategy

1. Use PostgreSQL for metadata integration tests only.
2. Use MinIO for raw object integration tests only.
3. Keep repository tests focused on metadata, transactions, constraints, and query semantics.
4. Keep storage tests focused on object upload, download, and delete.
5. Use service integration tests for workflows that require both PostgreSQL and MinIO.
6. Explicitly test partial failure states:
   - MinIO upload succeeds and DB insert fails.
   - MinIO delete fails during permanent delete.
   - MinIO delete succeeds and DB hard delete fails.
   - Folder permanent delete partially fails.
7. Integration tests that require unavailable local services must be skipped with an explicit reason; they must not silently pass.

### 3.5 Mocking and Test Data Strategy

| Fixture type | Strategy |
|---|---|
| Users | Use deterministic test users such as `owner@example.com`, `other@example.com`, and `viewer@example.com`. |
| Passwords | Use safe test-only values and assert that plaintext passwords are never stored or logged. |
| JWTs | Use deterministic test signing config only in tests; include valid, expired, malformed, and wrong-signature tokens. |
| Refresh tokens | Use token fixtures that represent valid, expired, revoked, replayed, and wrong-user states. Store only hashes in repository assertions. |
| Files | Use small stream fixtures, large synthetic stream fixtures, MIME mismatch fixtures, empty files, duplicate names, and path traversal names. |
| Folders | Use root folders, nested folders, deleted parents, cycle-attempt targets, and duplicate active sibling names. |
| MinIO objects | Use generated opaque object keys; never assert user-readable object key structure. |
| Share links | Use valid, expired, revoked, permission-mismatched, missing, and inaccessible-resource share tokens. |
| Logs | Use capture sinks or fake loggers that allow redaction assertions without writing secrets. |

### 3.6 Error and Edge-Case Strategy

- Invalid input must return safe `400` errors.
- Missing or invalid access tokens must return `401` on protected endpoints.
- Unauthorized private resource access should use safe not-found behavior where required.
- Duplicate active sibling names must return `409`.
- Stale versions must return `409`.
- Oversized upload requests must return `413` before body consumption.
- Unexpected errors must return safe `500` responses without internal details.
- Share-link validation failures must always return the same generic `404` body.
- Logs must not reveal passwords, access tokens, refresh tokens, share tokens, Authorization headers, MinIO credentials, object keys, raw URLs containing sensitive tokens, or sensitive query strings.

---

## 4. High-Risk Coverage Matrix

| High-risk item | Test IDs |
|---|---|
| User registration | AUTH-TC-001, AUTH-TC-002 |
| Login | AUTH-TC-003 |
| JWT validation | AUTH-TC-004, AUTH-TC-005 |
| Refresh token rotation | AUTH-TC-006 |
| Revoked refresh token replay detection | AUTH-TC-007 |
| Logout token revocation | AUTH-TC-009 |
| Streaming file upload without full-memory buffering | FILE-TC-001 |
| Upload file size limit before reading request body | FILE-TC-002 |
| Server-side MIME detection | FILE-TC-003 |
| Opaque MinIO object key generation | FILE-TC-004 |
| Streaming file download | FILE-TC-008 |
| File rename with optimistic locking | FILE-TC-010 |
| File move with authorization checks | FILE-TC-012 |
| Folder cascade soft-delete | FOLD-TC-005 |
| Recycle bin listing | BIN-TC-001 |
| Recycle bin restore with original parent structure | BIN-TC-002 |
| Permanent delete with MinIO cleanup success | BIN-TC-004 |
| Permanent delete with MinIO cleanup failure | BIN-TC-005 |
| Share link creation | SHARE-TC-001 |
| Share link expiry | SHARE-TC-003 |
| Share link revocation | SHARE-TC-004 |
| Anonymous share-link access | SHARE-TC-005, SHARE-TC-006 |
| Generic 404 response for invalid, expired, or revoked share tokens | SHARE-TC-007 |
| Sensitive data redaction in logs | SEC-TC-001, SHARE-TC-009 |

---

## 5. Core Test Cases

### 5.1 Authentication and User Management

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| AUTH-TC-001 | Authentication and User Management | P0 | API, Unit | No user exists for `owner@example.com`. | 1. Submit register request with valid email and password.<br>2. Read returned user DTO. | Response is `201`; user DTO includes id, email, display name, and creation time; no password hash is returned. | AUTH-01, AUTH-07, NFR-SEC-03 | Detailed Design §8.2, §8.7, §8.11 | Also assert normalized email where applicable. |
| AUTH-TC-002 | Authentication and User Management | P0 | Unit, Integration | User already exists for target email. | 1. Attempt duplicate registration.<br>2. Inspect DB user row if integration test is available. | Response maps to `409 DUPLICATE`; password is stored as a hash only; plaintext password never appears in DB or logs. | AUTH-01, AUTH-07, AUTH-08, NFR-SEC-02, NFR-SEC-03 | Detailed Design §8.4, §8.5, §8.7, §8.11 | Use safe log capture to assert redaction. |
| AUTH-TC-003 | Authentication and User Management | P0 | API, Unit | Registered user exists with hashed password. | 1. Submit valid login.<br>2. Submit login with wrong password.<br>3. Submit login for unknown email. | Valid login returns access token, refresh token, and expiry; invalid login returns safe failure without revealing whether email exists. | AUTH-02, AUTH-08, NFR-SEC-02, NFR-SEC-11 | Detailed Design §8.2, §8.7, §8.11 | Refresh token plaintext must not be logged. |
| AUTH-TC-004 | Authentication and User Management | P0 | API, Security | Protected route exists, such as `GET /api/v1/user/me`. | 1. Call protected route without JWT.<br>2. Call with valid JWT.<br>3. Call with malformed JWT. | Missing token returns `401 MISSING_TOKEN`; valid token succeeds; malformed token returns `401 INVALID_TOKEN`. | AUTH-03, AUTH-06, NFR-SEC-06 | Detailed Design §5.1, §6.9, §8.2, §8.8 | Applies to every protected API. |
| AUTH-TC-005 | Authentication and User Management | P0 | API, Security | Test JWT signing config is available. | 1. Call protected route with expired JWT.<br>2. Call with token signed by wrong key.<br>3. Call with wrong subject format. | All invalid tokens are rejected with `401`; no protected business handler runs. | AUTH-03, AUTH-06, NFR-SEC-06 | Detailed Design §6.9, §8.8 | Do not log the token value. |
| AUTH-TC-006 | Authentication and User Management | P1 | Unit, Integration, API | User has one active refresh token. | 1. Submit refresh request with valid refresh token.<br>2. Inspect old and new token states. | Request succeeds; old refresh token is revoked; new refresh token and new access token are issued; DB stores only token hash. | AUTH-04, AUTH-09 | Detailed Design §7.2, §8.2, §8.7, §8.11 | Rotation must be transactional. |
| AUTH-TC-007 | Authentication and User Management | P1 | Unit, Integration, Security | A refresh token was already revoked by a prior refresh. | 1. Re-submit the revoked refresh token.<br>2. Inspect active refresh tokens for the user. | Request returns `401 TOKEN_REVOKED`; replay is detected; all active refresh tokens for that user are revoked. | AUTH-09 | Detailed Design §6.8, §8.4, §8.7, §8.11 | Must not reveal token hash or token value. |
| AUTH-TC-008 | Authentication and User Management | P1 | Integration, Regression | User has one active refresh token. | 1. Send two concurrent refresh requests with the same token.<br>2. Inspect repository state. | At most one request succeeds; the token family remains in a safe state; no duplicate active replacement from the same old token. | AUTH-09, NFR-SEC-13 | Detailed Design §6.8, §8.7, §8.11 | Use DB transaction assertions when available. |
| AUTH-TC-009 | Authentication and User Management | P1 | API, Integration | Authenticated user has active refresh token. | 1. Submit logout with that refresh token.<br>2. Attempt refresh with the same token. | Logout returns `204`; token is revoked; later refresh fails. | AUTH-05, AUTH-09 | Detailed Design §8.2, §8.7, §8.11 | Also reject logout for token owned by another user. |
| AUTH-TC-010 | Authentication and User Management | P0 | API, Security | User has valid JWT. | 1. Call `GET /api/v1/user/me` with valid JWT.<br>2. Inspect response fields. | Response returns only public profile fields; no password hash, refresh token metadata, or internal values are returned. | AUTH-06, NFR-SEC-01 | Detailed Design §5.2, §8.2, §8.7, §8.11 | Also covered in frontend session flow. |

### 5.2 File Management

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| FILE-TC-001 | File Management | P0 | Unit, API, Security | Authenticated owner exists; fake reader can detect full reads. | 1. Upload file with a controlled streaming reader.<br>2. Instrument reader to fail if full-buffer read behavior is attempted.<br>3. Verify storage receives an `io.Reader` stream. | Upload path streams content and does not load the full file into memory. | FILE-01, FILE-02, NFR-PERF-01 | Detailed Design §6.3, §6.5, §9.3, §9.7, §9.9 | Static review may additionally forbid `io.ReadAll` and `ioutil.ReadAll` in production upload paths. |
| FILE-TC-002 | File Management | P0 | Handler, API, Security | Upload limit is configured; request body reader records whether it was consumed. | 1. Send multipart upload request with `Content-Length` above limit.<br>2. Assert handler response.<br>3. Assert body was not consumed beyond pre-read validation. | Response is `413 FILE_TOO_LARGE`; request body is rejected before reading file content. | FILE-03, NFR-SEC-07, NFR-SCALE-04 | Detailed Design §6.3, §6.5, §6.9, §9.3, §9.9 | This is a must-have regression test for upload handler changes. |
| FILE-TC-003 | File Management | P0 | Unit, API, Security | Upload body bytes represent one MIME type while client header claims another. | 1. Upload file with misleading client MIME header.<br>2. Inspect stored metadata.<br>3. Inspect response DTO. | Stored MIME type is derived from server-side sniffing, not from client-provided header. | FILE-06, NFR-SEC-08 | Detailed Design §6.3, §6.5, §9.4, §9.9 | Only the sniffed prefix may be buffered. |
| FILE-TC-004 | File Management | P0 | Unit, Integration, Security | User uploads file named with user-identifiable or path-like string. | 1. Upload file named `owner/Documents/report.pdf`.<br>2. Inspect storage upload object key.<br>3. Inspect response DTO. | Object key is generated internally, opaque, unique, and does not contain user ID, folder name, or filename; response does not expose key. | FILE-04, FILE-05, SHARE-06 | Detailed Design §6.4, §7.4, §9.4, §9.9 | Never assert a human-readable key format. |
| FILE-TC-005 | File Management | P0 | Unit, Integration, API | Authenticated owner uploads known byte stream. | 1. Upload known bytes.<br>2. Compare stored SHA256 metadata to expected hash.<br>3. Inspect response. | SHA256 hash is computed during streaming and stored; response exposes safe metadata only. | FILE-07, NFR-REL-02 | Detailed Design §6.6, §7.4, §9.7, §9.9 | Do not perform a second read pass over non-seekable streams. |
| FILE-TC-006 | File Management | P0 | Unit, Integration, Regression | Storage upload succeeds; repository insert is configured to fail. | 1. Upload file through service.<br>2. Trigger DB insert failure.<br>3. Verify cleanup call to storage. | Service attempts MinIO cleanup; returns safe error; cleanup failure, if any, is logged safely. | NFR-REL-01, FILE-02 | Detailed Design §6.7, §9.7, §9.9 | No DB metadata should be committed for failed upload. |
| FILE-TC-007 | File Management | P0 | API | Authenticated owner sends malformed multipart request. | 1. Send upload without file part.<br>2. Send upload with more than one unexpected file part if handler policy supports one file only. | Request is rejected with safe validation error; no MinIO upload or file metadata insert occurs. | FILE-01, FILE-12 | Detailed Design §5.1, §9.3, §9.9 | Keep error message user-friendly. |
| FILE-TC-008 | File Management | P0 | Unit, API, Integration | Active file exists and storage has object. | 1. Download file as owner.<br>2. Instrument storage reader and response writer.<br>3. Inspect response headers and body streaming behavior. | Service checks authorization before opening storage stream; response streams through writer and does not expose object key. | FILE-08, FILE-12, NFR-PERF-01 | Detailed Design §6.3, §9.7, §9.8, §9.9 | Must support large file streaming. |
| FILE-TC-009 | File Management | P0 | API, Security | Active file exists for owner. | 1. Download without JWT.<br>2. Download as another user.<br>3. Download after file is soft-deleted. | Missing JWT returns `401`; cross-user and deleted-file access return safe not-found style behavior where required; storage is not opened for unauthorized requests. | FILE-08, FILE-11, FILE-12, SHARE-07, NFR-SEC-11 | Detailed Design §6.1, §6.9, §9.8, §9.9 | Prevent resource enumeration. |
| FILE-TC-010 | File Management | P1 | Unit, Integration, API | Active file exists with version `1`. | 1. Rename with version `1`.<br>2. Rename again with stale version `1`. | First rename succeeds and increments version; stale update returns `409 CONFLICT`. | FILE-09, NFR-SEC-13 | Detailed Design §6.8, §9.7, §9.9, §14.3 | Required high-risk optimistic locking case. |
| FILE-TC-011 | File Management | P1 | Unit, API, Security | Active file exists in folder. | 1. Rename to invalid empty name.<br>2. Rename to path traversal-like name.<br>3. Rename to duplicate active sibling name. | Invalid and traversal names are rejected; duplicate active sibling returns `409 DUPLICATE`; no storage operation occurs. | FILE-09, FILE-13, NFR-SEC-05 | Detailed Design §6.9, §9.7, §9.9 | User-supplied names are display metadata only. |
| FILE-TC-012 | File Management | P1 | Unit, API, Security | Owner file exists; target folders exist for owner and another user. | 1. Move owner file to owner folder.<br>2. Move owner file to another user's folder.<br>3. Move another user's file. | Valid move succeeds; cross-user target and cross-user file attempts are rejected; authorization is checked before update. | FILE-10, FILE-12, NFR-SEC-01 | Detailed Design §6.1, §9.7, §9.8, §9.9 | Required high-risk authorization case. |
| FILE-TC-013 | File Management | P1 | Unit, Integration | File exists; target folder may be deleted; file version can be stale. | 1. Move with stale version.<br>2. Move to deleted folder. | Stale move returns `409 CONFLICT`; deleted target is rejected; file remains in original parent. | FILE-10, FILE-12, NFR-SEC-13 | Detailed Design §6.8, §9.7, §9.9 | Also validates transactional consistency. |
| FILE-TC-014 | File Management | P0 | Unit, Integration, API | Active file exists with MinIO object. | 1. Soft-delete file.<br>2. Inspect DB metadata.<br>3. Inspect MinIO object state. | File is marked deleted and appears in recycle bin; MinIO object remains intact. | FILE-11, BIN-01 | Detailed Design §6.7, §9.7, §11.7 | Soft delete must not remove object content. |

### 5.3 Folder Management

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| FOLD-TC-001 | Folder Management | P0 | Unit, API, Integration | Authenticated user exists. | 1. Create root folder.<br>2. Create folder under active parent owned by same user. | Both folders are created with correct parent metadata and safe DTOs. | FOLD-01 | Detailed Design §10.2, §10.7, §10.10 | No MinIO call is expected for folder creation. |
| FOLD-TC-002 | Folder Management | P0 | Unit, API, Security | Parent folder exists. | 1. Create invalid folder name.<br>2. Create path traversal-like folder name.<br>3. Create duplicate active sibling. | Invalid and traversal names are rejected; duplicate active sibling returns `409 DUPLICATE`. | FOLD-01, FILE-13, NFR-SEC-05 | Detailed Design §10.3, §10.7, §10.10 | Applies to rename as well. |
| FOLD-TC-003 | Folder Management | P1 | Unit, Integration, API | Active folder exists with version `1`. | 1. Rename with correct version.<br>2. Rename with stale version. | Correct version succeeds and increments version; stale version returns `409 CONFLICT`. | FOLD-02, NFR-SEC-13 | Detailed Design §6.8, §10.7, §10.10 | Mirrors file optimistic locking behavior. |
| FOLD-TC-004 | Folder Management | P1 | Unit, Integration, Security | Nested folder tree exists. | 1. Move folder to root.<br>2. Move folder under another owned folder.<br>3. Attempt move under descendant.<br>4. Attempt move to another user's or deleted parent. | Valid moves succeed; cycle, cross-user parent, and deleted parent are rejected. | FOLD-02, FOLD-06, NFR-SEC-01 | Detailed Design §6.8, §10.7, §10.8, §10.10 | Does not implement ACL inheritance. |
| FOLD-TC-005 | Folder Management | P0 | Unit, Integration, API, Regression | Folder tree contains subfolders and files. | 1. Soft-delete parent folder.<br>2. Inspect subtree metadata.<br>3. Inspect MinIO object state. | Folder, child folders, and contained files are marked deleted in DB; no MinIO object is deleted. | FOLD-03, FOLD-04, BIN-08 | Detailed Design §10.7, §10.9, §10.10, §14.4 | Required high-risk cascade soft-delete case. |
| FOLD-TC-006 | Folder Management | P0 | Integration, Regression | Folder cascade update is configured to fail midway. | 1. Trigger cascade soft-delete.<br>2. Force repository failure.<br>3. Inspect transaction state. | Transaction rolls back; subtree is not partially marked deleted. | FOLD-03, FOLD-04, NFR-REL-01 | Detailed Design §10.9, §10.10 | No MinIO cleanup is involved for soft delete. |

### 5.4 Recycle Bin

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| BIN-TC-001 | Recycle Bin | P0 | API, Integration, Security | Deleted files and folders exist for owner and another user. | 1. List recycle bin as owner.<br>2. List as another user. | Owner sees only own deleted items; active items and other users' deleted items are excluded. | BIN-02, BIN-06 | Detailed Design §11.2, §11.7, §11.8, §11.10 | Required high-risk listing case. |
| BIN-TC-002 | Recycle Bin | P0 | Unit, Integration, API, Regression | Deleted file has an original active parent structure. | 1. Restore file from recycle bin.<br>2. Inspect file metadata and parent folder state. | File is restored to its original parent structure as much as possible and is removed from recycle bin listing. | BIN-03, FOLD-05 | Detailed Design §11.7, §11.9, §11.10, §14.5 | Required high-risk original-parent case. |
| BIN-TC-003 | Recycle Bin | P0 | Unit, Integration | Deleted item has missing, permanently removed, deleted, or duplicate-name parent situation. | 1. Restore deleted item with missing parent.<br>2. Restore item with duplicate active name.<br>3. Restore folder with deleted ancestor chain. | Missing parent falls back to approved safe location such as root; duplicate active name is rejected; folder subtree is restored consistently. | BIN-03, FOLD-05 | Detailed Design §11.7, §11.9, §11.10 | Preserve directory structure as much as possible. |
| BIN-TC-004 | Recycle Bin | P0 | Unit, Integration, API, Regression | Deleted file exists and MinIO delete will succeed; explicit confirmation is supplied. | 1. Permanently delete file.<br>2. Inspect storage delete call.<br>3. Inspect DB row. | MinIO object is deleted first; DB row is hard-deleted after storage success; response is safe. | BIN-04, NFR-REL-04 | Detailed Design §6.7, §11.7, §11.10, §14.6 | Required high-risk cleanup success case. |
| BIN-TC-005 | Recycle Bin | P0 | Unit, Integration, Regression | Deleted file exists and MinIO delete is configured to fail. | 1. Permanently delete file.<br>2. Force storage delete failure.<br>3. Inspect DB row. | Operation returns safe failure; DB metadata remains deleted and retryable; log is redacted. | BIN-05, NFR-REL-04 | Detailed Design §6.7, §11.7, §11.10, §14.6 | Required high-risk cleanup failure case. |
| BIN-TC-006 | Recycle Bin | P0 | Integration, Regression | MinIO delete succeeds but DB hard delete fails. | 1. Force DB hard-delete failure after storage success.<br>2. Retry permanent delete. | Deleted metadata remains retryable; storage delete is idempotent; retry reaches consistent final state. | BIN-04, BIN-05, NFR-REL-04 | Detailed Design §6.7, §11.7, §11.10 | Prevent inconsistent visible active state. |
| BIN-TC-007 | Recycle Bin | P0 | Unit, Integration | Deleted folder subtree contains multiple files; one MinIO delete fails. | 1. Permanently delete folder.<br>2. Force one object delete failure.<br>3. Inspect DB and storage state. | Partial failure is handled without orphaning metadata or pretending full success; retry remains possible. | BIN-04, BIN-05, FOLD-04 | Detailed Design §6.7, §11.7, §11.10, §14.6 | The exact atomicity choice must match implementation policy. |
| BIN-TC-008 | Recycle Bin | P1 | Integration, Regression | Expired recycle-bin items exist; one MinIO delete will fail. | 1. Run cleanup mechanism.<br>2. Inspect per-item results. | Cleanup logs safe error, continues with later items, and leaves failed item retryable. | BIN-07, NFR-REL-03 | Detailed Design §11.7, §11.10 | Scheduling implementation is not designed here. |
| BIN-TC-009 | Recycle Bin | P0 | API, Security, Frontend | Deleted item exists for owner. | 1. Attempt permanent delete without explicit confirmation.<br>2. Attempt restore or delete as another user. | Missing confirmation is rejected; cross-user recycle operations are rejected with safe behavior. | BIN-04, BIN-06 | Detailed Design §11.8, §11.10, §13.4 | Frontend must require explicit confirmation. |

### 5.5 File Sharing

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| SHARE-TC-001 | File Sharing | P1 | Unit, API, Integration | Owner has active file or folder. | 1. Create share link for owned file.<br>2. Create share link for owned folder.<br>3. Inspect DTO. | Share link is created with resource type, resource id, permission, expiry, revoked state, and owner-shareable URL; token is not logged. | SHARE-01, SHARE-03, SHARE-08 | Detailed Design §12.2, §12.7, §12.11 | Required high-risk creation case. |
| SHARE-TC-002 | File Sharing | P1 | Unit, API, Security | Active and deleted resources exist across users. | 1. Attempt share creation for another user's resource.<br>2. Attempt share creation for deleted resource.<br>3. Attempt invalid TTL or permission. | Requests are rejected safely; no share link row is created. | SHARE-01, FILE-12, NFR-SEC-01 | Detailed Design §12.7, §12.8, §12.11 | Validate both file and folder resources. |
| SHARE-TC-003 | File Sharing | P1 | Unit, API, Security | Share link exists with expired `expires_at`. | 1. Access expired share token anonymously.<br>2. Compare response body with other share failure cases. | Expired token returns the same generic `404` response as other share-token failures. | SHARE-03, SHARE-05, SHARE-07 | Detailed Design §6.9, §12.7, §12.11 | Required high-risk expiry case. |
| SHARE-TC-004 | File Sharing | P1 | Unit, API, Integration, Security | Active share link exists for owner. | 1. Revoke share as owner.<br>2. Access revoked token anonymously.<br>3. Attempt revoke as another user. | Owner revoke succeeds; revoked token returns generic `404`; cross-user revoke is rejected. | SHARE-02, SHARE-05 | Detailed Design §12.7, §12.8, §12.11 | Required high-risk revocation case. |
| SHARE-TC-005 | File Sharing | P0 | API, Security, Regression | Valid non-expired, non-revoked share token exists. | 1. Access `GET /api/v1/shares/:token` without JWT.<br>2. Inspect response DTO. | Anonymous access succeeds only through share-token validation and returns safe DTOs. | SHARE-05, NFR-SEC-06 | Detailed Design §12.2, §12.7, §12.11, §14.7 | Required high-risk anonymous access case. |
| SHARE-TC-006 | File Sharing | P1 | API, Security | Share links exist with `preview` and `download` permissions. | 1. Attempt download through preview-only share.<br>2. Attempt download through download share if endpoint behavior is implemented in API. | Preview-only scope does not allow explicit download; download scope streams content when allowed. | SHARE-04, SHARE-05, FILE-08 | Detailed Design §12.7, §12.8, §12.11 | Browser preview does not guarantee copy protection. |
| SHARE-TC-007 | File Sharing | P0 | API, Security, Regression | Missing, invalid, expired, revoked, mismatched-scope, and inaccessible-resource share-token states exist. | 1. Access each token state anonymously.<br>2. Compare status, code, and message. | Every failure returns the same generic `404` response body and no state-specific detail. | SHARE-05, SHARE-07, NFR-SEC-11 | Detailed Design §6.9, §12.7, §12.11 | Required high-risk generic 404 case. |
| SHARE-TC-008 | File Sharing | P0 | API, Security | Valid share response is available for file and folder. | 1. Create share link.<br>2. Access share link.<br>3. Inspect all response fields. | No MinIO object key, bucket URL, credentials, internal storage path, or token hash is exposed. | SHARE-06, NFR-SEC-02 | Detailed Design §5.2, §6.2, §12.7, §12.11 | Applies to owner and anonymous responses. |
| SHARE-TC-009 | File Sharing | P0 | Security, Regression | Logging capture is available. | 1. Create share link.<br>2. Access share token route.<br>3. Trigger share-token failure.<br>4. Inspect logs. | Logs use sanitized route pattern and do not contain share token, raw URL, query token, MinIO object key, or credentials. | SHARE-08, NFR-SEC-02, NFR-SEC-14 | Detailed Design §6.2, §12.11 | Required redaction case. |

### 5.6 File Preview Pending Scope

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| PREV-TC-001 | File Preview | P0 | Review, Regression | Preview decision is still pending. | 1. Review API contract changes before implementation.<br>2. Review dependency changes before implementation.<br>3. Review test plan additions. | No server-side preview conversion service, OCR, ffmpeg, LibreOffice, preview engine, or preview implementation test is added until the pending decision is resolved. | PREV-01, PREV-02, PREV-03 | Detailed Design §3, §18; Overview Design §5.6 | This is a scope guard only, not a preview implementation test. |
| PREV-TC-002 | File Preview | P2 | Review | Preview decision is approved in the future. | 1. Reopen test design.<br>2. Add range-request and browser-native preview tests only after approval.<br>3. Confirm no full-file buffering. | Future preview tests are designed only after decision approval and must preserve streaming rules. | PREV-01, PREV-02, PREV-03 | Detailed Design §6.3, §18; Overview Design §5.6 | Do not implement or design conversion tests now. |

### 5.7 Authorization and Security

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| SEC-TC-001 | Authorization and Security | P0 | Security, Regression | Log capture is available. | 1. Exercise auth, upload, download, share, and error paths with sensitive values.<br>2. Inspect logs. | Logs do not contain passwords, access tokens, refresh tokens, token hashes, share tokens, Authorization headers, MinIO credentials, object keys when avoidable, raw sensitive URLs, or sensitive query strings. | AUTH-08, SHARE-08, NFR-SEC-02, NFR-SEC-14 | Detailed Design §6.2, §8.11, §12.11 | Required high-risk redaction case. |
| SEC-TC-002 | Authorization and Security | P0 | API, Security | Route table is available. | 1. Call public endpoints without JWT.<br>2. Call protected endpoints without JWT.<br>3. Call refresh endpoint without access JWT but with valid refresh token. | Only register, login, refresh, and share access are public; all other APIs require valid JWT. | AUTH-06, NFR-SEC-06 | Detailed Design §5.1, §8.2, §12.2 | Refresh validates refresh token server-side. |
| SEC-TC-003 | Authorization and Security | P0 | API, Security | Filename and folder name inputs are accepted by relevant endpoints. | 1. Submit names containing `../`, path separators, encoded traversal, or absolute paths.<br>2. Submit same patterns in file rename and folder create/rename. | Inputs are rejected or sanitized according to approved rules; no object key or storage path uses user input. | FILE-13, NFR-SEC-05 | Detailed Design §6.3, §9.9, §10.10 | Avoid platform-specific path assumptions. |
| SEC-TC-004 | Authorization and Security | P1 | API, Security | Private resources exist for owner and another user. | 1. Access another user's file, folder, recycle item, and share management endpoint.<br>2. Compare errors to missing-resource behavior where required. | Unauthorized users cannot determine whether private resources exist through error messages or state-specific details. | FILE-12, SHARE-07, NFR-SEC-11 | Detailed Design §6.1, §6.9, §9.8, §10.8, §11.8, §12.8 | Timing side-channel tests may be limited to coarse regression checks. |
| SEC-TC-005 | Authorization and Security | P0 | Repository, API, Security | Repository integration setup is available. | 1. Submit SQL-like strings in email, names, search or list parameters where applicable.<br>2. Inspect DB behavior. | Inputs are parameterized and cannot alter SQL semantics; service returns safe validation or normal data response. | NFR-SEC-04 | Detailed Design §15 | This test validates repository query safety without adding a new dependency. |
| SEC-TC-006 | Authorization and Security | P1 | API, Security | Rate limiting middleware is implemented. | 1. Send repeated login attempts.<br>2. Send repeated upload or download requests when endpoint limits are configured. | Auth rate limit returns `429 RATE_LIMITED`; upload/download limits are configurable and do not expose internal state. | NFR-SEC-09, NFR-SEC-12 | Detailed Design §6.9; Architecture Design §8 | If rate limiting is not implemented yet, keep as pending implementation test. |

### 5.8 Error Handling

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| ERR-TC-001 | Error Handling | P0 | API, Unit, Regression | Domain error fixtures are available. | 1. Trigger invalid request, validation failure, missing token, invalid token, duplicate, conflict, file too large, not found, rate limited, and internal error.<br>2. Inspect HTTP response. | Error body follows `{ "error": { "code": "...", "message": "..." } }`; status and code match mapping. | NFR-UI-03 | Detailed Design §5.1, §6.9 | Internal errors must not leak stack traces. |
| ERR-TC-002 | Error Handling | P1 | API | List endpoints exist with pagination. | 1. Request negative offset.<br>2. Request limit above max.<br>3. Request valid pagination. | Invalid pagination is rejected or normalized per API convention; valid list response follows `{ "data": [...], "total": N }`. | NFR-PERF-02 | Detailed Design §5.1, §9.2, §11.2 | Max limit is 100 by design. |
| ERR-TC-003 | Error Handling | P0 | Unit, API, Security | Unexpected service error can be injected. | 1. Force repository or storage unexpected error.<br>2. Inspect response and log. | Response is safe `500 INTERNAL_ERROR`; log contains safe error category and no secrets. | NFR-SEC-02, NFR-UI-03 | Detailed Design §6.2, §6.9, §12 | Applies to all modules. |

### 5.9 Frontend UI Flows

| Test ID | Module | Priority | Test type | Preconditions | Steps | Expected result | Related requirement ID | Related detailed design section | Notes / edge cases |
|---|---|---:|---|---|---|---|---|---|---|
| FE-TC-001 | Frontend UI Flows | P0 | Frontend, Security | Frontend auth UI exists. | 1. Submit valid login form.<br>2. Inspect session state.<br>3. Trigger logout. | Access token is held only in in-memory state; logout clears session state and calls logout API when applicable. | AUTH-02, AUTH-05, NFR-SEC-02 | Detailed Design §13.2, §13.4 | Do not store JWT in persistent browser storage unless design changes. |
| FE-TC-002 | Frontend UI Flows | P0 | Frontend, API Client | API client exists. | 1. Send protected API request.<br>2. Send anonymous share request.<br>3. Inspect constructed headers. | Protected requests include bearer token; anonymous share access does not include JWT. | AUTH-06, SHARE-05, NFR-SEC-06 | Detailed Design §13.3, §13.4 | Prevent leaking authenticated context into public share access. |
| FE-TC-003 | Frontend UI Flows | P1 | Frontend | File and folder listing API can be mocked. | 1. Load folder view.<br>2. Render files and folders.<br>3. Handle empty, loading, and error states. | UI shows folder contents, pagination state where applicable, and user-friendly errors. | FILE-01, FOLD-01, NFR-UI-03 | Detailed Design §13.2, §13.4 | No internal IDs except safe resource IDs should be exposed in text. |
| FE-TC-004 | Frontend UI Flows | P1 | Frontend, API Client | Upload UI exists. | 1. Select file through picker.<br>2. Use drag-and-drop entry point if implemented.<br>3. Inspect API call and progress state. | Upload sends multipart form data, shows progress, and handles success and error states. | FILE-01, NFR-UI-02, NFR-UI-04 | Detailed Design §13.2, §13.4 | Drag-and-drop is P2 but can be tested when implemented. |
| FE-TC-005 | Frontend UI Flows | P1 | Frontend | File or folder version is shown in mocked API data. | 1. Rename or move item.<br>2. Mock `409 CONFLICT` response.<br>3. Inspect UI behavior. | UI sends current `version`; conflict prompts refresh or retry behavior without losing user context. | FILE-09, FILE-10, FOLD-02, NFR-SEC-13 | Detailed Design §13.4 | No optimistic-lock bypass in UI. |
| FE-TC-006 | Frontend UI Flows | P0 | Frontend, Regression | Recycle bin UI exists. | 1. Soft-delete item.<br>2. Open recycle bin.<br>3. Restore item.<br>4. Attempt permanent delete. | Deleted item appears in recycle bin; restore updates UI; permanent delete requires explicit confirmation. | BIN-01, BIN-02, BIN-03, BIN-04 | Detailed Design §13.4 | Confirmation must be visible and deliberate. |
| FE-TC-007 | Frontend UI Flows | P1 | Frontend, Security | Share UI and anonymous share page exist. | 1. Create share link.<br>2. Revoke share link.<br>3. Open anonymous share route.<br>4. Mock generic 404. | UI presents safe share URL to owner, handles revocation, and displays generic not-found state without explaining token state. | SHARE-01, SHARE-02, SHARE-05 | Detailed Design §13.4 | Do not reveal expired vs revoked vs invalid token reason. |
| FE-TC-008 | Frontend UI Flows | P1 | Frontend | Responsive layout test setup exists. | 1. Render main pages at desktop and tablet viewport sizes.<br>2. Inspect core controls and layout. | Main flows remain usable on desktop and tablet. | NFR-UI-01 | Detailed Design §13.2 | Avoid pixel-perfect tests unless approved by frontend strategy. |

---

## 6. Regression Suite Selection

The default regression suite should include:

| Area | Required regression test IDs |
|---|---|
| Authentication | AUTH-TC-001, AUTH-TC-003, AUTH-TC-004, AUTH-TC-006, AUTH-TC-007, AUTH-TC-009 |
| File upload and download | FILE-TC-001, FILE-TC-002, FILE-TC-003, FILE-TC-004, FILE-TC-006, FILE-TC-008, FILE-TC-009 |
| File and folder updates | FILE-TC-010, FILE-TC-012, FOLD-TC-003, FOLD-TC-005 |
| Recycle bin | BIN-TC-001, BIN-TC-002, BIN-TC-004, BIN-TC-005, BIN-TC-009 |
| Sharing | SHARE-TC-001, SHARE-TC-003, SHARE-TC-004, SHARE-TC-005, SHARE-TC-007, SHARE-TC-009 |
| Security and errors | SEC-TC-001, SEC-TC-002, SEC-TC-003, SEC-TC-004, ERR-TC-001, ERR-TC-003 |
| Frontend | FE-TC-001, FE-TC-002, FE-TC-004, FE-TC-005, FE-TC-006, FE-TC-007 |
| Pending scope guards | PREV-TC-001 |

---

## 7. Coverage Summary

| Module | Coverage included |
|---|---|
| Authentication and User Management | Registration, duplicate registration, password hashing, login, JWT validation, refresh token rotation, replay detection, concurrent refresh, logout revocation, current user profile, and token redaction. |
| File Management | Streaming upload, upload size pre-check, MIME detection, SHA256, opaque object keys, upload partial failure cleanup, streaming download, authorization, rename, move, optimistic locking, path traversal rejection, and soft delete. |
| Folder Management | Create, rename, move, duplicate name handling, path traversal rejection, cycle rejection, cross-user parent rejection, deleted parent rejection, cascade soft-delete, and rollback. |
| File Sharing | Creation, TTL validation, expiry, revocation, anonymous access, permission scope, generic 404 failure equivalence, non-exposure of storage internals, and share-token redaction. |
| Recycle Bin | Listing, restore with original parent structure, missing-parent restore behavior, duplicate restore conflict, permanent delete success, permanent delete failure, folder permanent delete partial failure, scheduled cleanup, permissions, and explicit confirmation. |
| File Preview | Pending-only scope guard. No server-side conversion, OCR, ffmpeg, LibreOffice, or implementation tests are designed. |
| Authorization and Security | Public endpoint whitelist, JWT enforcement, path traversal rejection, private resource non-enumeration, SQL injection resistance, rate limiting where implemented, and sensitive data redaction. |
| Error Handling | Standard error response shape, domain-to-HTTP mapping, pagination validation, and safe internal error handling. |
| Frontend UI Flows | Login/session, API client headers, folder browsing, upload/progress, conflict handling, recycle bin flow, share flow, anonymous share error handling, and responsive usability. |

---

## 8. Risk Areas

1. Upload and download streaming may regress if future code uses full-body buffering.
2. Upload size limits must be enforced before body consumption, which requires careful handler tests.
3. MIME detection must avoid trusting client-provided headers while also avoiding full-file buffering.
4. MinIO and PostgreSQL writes are not atomic, so partial failure tests are critical.
5. Refresh token rotation must be transactional to prevent replay and concurrent refresh issues.
6. Share-link failure responses must stay indistinguishable to prevent token or resource-state enumeration.
7. Recycle bin restore and permanent delete can create inconsistent states if parent structure, duplicate names, or MinIO failures are mishandled.
8. Folder cascade soft-delete must be transactional and must not remove MinIO objects.
9. Optimistic locking must be enforced consistently across file and folder rename or move operations.
10. Logging middleware must avoid raw URLs and route parameters because share tokens may be path parameters.
11. Frontend tests must not rely on internal storage details or expose token-specific failure reasons.
12. Pending preview and folder permission inheritance decisions must not accidentally expand scope.

---

## 9. Open Questions

1. Which approved PostgreSQL driver and MinIO/S3 client will be used during implementation?
2. Which approved test runner and assertion approach will be used for frontend tests if the initial React project does not already include one?
3. What is the exact configured maximum upload size for local development and default deployment?
4. What is the approved maximum share-link TTL and whether non-expiring links are allowed?
5. What is the exact API behavior for share download scope if the final OpenAPI separates preview metadata and download access?
6. What is the final restore policy when the original parent path conflicts with an active item name?
7. What atomicity policy will implementation choose for folder permanent delete when only some MinIO objects are removed successfully?
8. What rate-limit thresholds will be configured for auth, upload, and download endpoints?
9. How will integration tests discover local PostgreSQL and MinIO endpoints without introducing new tooling?
10. When the preview decision is resolved, which browser-native formats and range-request semantics will become testable?

---

## 10. Recommended Next Step Before Implementation

Before writing any business code, test code, migration SQL, or OpenAPI changes:

1. Review this `test_design.md` against `requirements.md`, `architecture_design.md`, `overview_design.md`, `detailed_design.md`, and `AGENTS.md`.
2. Resolve or explicitly defer the open questions that affect test expectations.
3. Confirm the approved dependency choices for PostgreSQL, MinIO/S3, JWT, password hashing, and any frontend test tooling.
4. Update `api/openapi.yaml` first for any request or response shape that implementation will use.
5. Implement the smallest vertical slice with matching tests, starting with authentication or file upload only after the API contract is ready.
6. Run only the verification commands that exist at that time, and explicitly report any skipped checks with reasons.
