# NuoNetDisk — Agent Instructions

## Project Status

Greenfield project. Repository is empty. Build from scratch.

## Git Operations (Blocked)

This environment cannot execute git commands (commit, push, branch, rebase, etc.). Do not attempt any git operations.

## Language Policy

All text in this project must be English only:
- Code comments, documentation, commit messages, PR descriptions
- API descriptions and error messages
- Variable / function / test names
- Seed data and sample data (unless business requires non-English content)

No Chinese in any project file.

## Tech Stack (Hard Decisions)

| Layer | Choice |
|---|---|
| Backend language | **Go** |
| Backend framework | **Gin** |
| Frontend | **React** + **TypeScript** |
| Metadata DB | **PostgreSQL** |
| File storage | **MinIO** (S3-compatible) |
| Auth | **JWT** |

Start with a backend monolith. Do not introduce microservices.
File preview approach is pending — do not introduce server-side conversion (LibreOffice / ffmpeg / OCR) without approval.

## Architecture

- Frontend-backend separation: `frontend/` and `backend/` directories.
- Backend layout: `cmd/server/main.go`, `internal/`, `pkg/`.
- Frontend: standard React + TypeScript project (Vite).
- API contracts in `api/openapi.yaml`. Update before writing handlers.
- Do not pre-create deep directory trees. Let structure grow with features.

## Verifying Changes

Run these before marking any task complete:

- **Backend changes**: `go vet ./...`, `go test ./...`, `gofmt -d .` (must produce no diffs)
- **Frontend changes**: `npm run typecheck`, `npm run lint`, `npm run test`
- **API changes**: Check OpenAPI spec is in sync
- **File/permission changes**: Must include tests

If a command does not exist yet, do not pretend it does. Set it up first, then verify.

## File Safety Rules (HARD REQUIREMENTS)

- Uploads and downloads MUST stream via `io.Reader` / `io.Writer`. Never `io.ReadAll`, `ioutil.ReadAll`, or equivalent.
- File previews must not load entire files into memory.
- Enforce file size limits during upload (before reading the body).
- User-supplied filenames must NEVER be used as MinIO object keys. Use internal IDs (UUIDs) or generated paths.
- MinIO object keys must be opaque — no user-identifiable or path-revealing strings.
- All file operations (upload, download, preview, rename, move, delete, restore, share) must check authorization.

## Database & Storage Boundary

- **PostgreSQL**: metadata only — users, files, folders, permissions, shares, recycle bin state.
- **MinIO**: raw file objects only.
- Do not store file contents in PostgreSQL (no bytea blobs).
- DB schema must include: object key, size, MIME type, hash (SHA256), owner, parent folder, permissions, delete state, timestamps.
- MinIO writes and DB transactions are NOT atomic. Handle partial failure explicitly (e.g. upload succeeds but DB insert fails → clean up MinIO object).

## Permission & Security Rules

- Every file/folder operation must authorize. No operation may skip permission checks.
- Share links must be revocable, expirable, and limited in scope.
- Unauthorized users must not be able to detect whether a private file exists (no timing or error-message leaks).
- Folder permission inheritance strategy must be designed explicitly — do not assume flat ownership.
- Never log JWT tokens, passwords, share tokens, or MinIO credentials.
- Prevent path traversal attacks on user-supplied filenames and paths.
- Do not trust client-supplied MIME types — detect server-side.
- Do not expose internal MinIO object keys or bucket URLs to the frontend.
- All authentication and authorization must happen server-side.
- Deletions are soft-delete by default. Permanent delete removes the MinIO object and requires explicit confirmation.
- Restore must preserve the original directory structure as much as possible.
- Deleting a folder must cascade to sub-files and sub-folders in both DB and MinIO operations.
- Permanent delete must handle MinIO cleanup failures gracefully.

## Recycle Bin Rules

- "Delete" means soft-delete. Move to recycle bin, set `deleted_at`.
- "Permanent delete" actually removes the MinIO object and the DB record.
- Restore from recycle bin must restore the original parent structure.
- Recycle bin operations still need permission checks.
- Scheduled cleanup of expired recycle bin items must handle partial MinIO delete failures.

## Dependency Policy

- No new dependency (runtime or dev) without explicit approval.
- Justify: what problem it solves, alternatives considered, maintenance status, security impact.
- Prefer Go standard library over external packages.
- Prefer well-established, actively maintained libraries for anything non-trivial.
- Applies equally to Go modules (`go.mod`), npm packages (`package.json`), and any other package manager.

## High-Risk Areas

These areas require extra caution and should have test coverage:

- File upload / download streaming
- Permission enforcement (every code path)
- Share link creation and validation
- Recycle bin (soft delete, restore, cascade)
- MinIO object deletion and cleanup after partial failure
- Database migrations
- JWT authentication middleware
- File preview (when implemented)

## Mandatory Skill

Every task (backend, frontend, API, config, docs) must load the `karpathy-guidelines` skill. Its rules — think before coding, simplicity first, surgical changes, goal-driven execution — apply globally.

## Developer Workflow

1. API contract first → write/update `api/openapi.yaml`
2. Implement backend handlers
3. Implement frontend UI
4. Verify (lint → typecheck → test → vet)
5. If changing request/response shape, update OpenAPI spec first
6. Do NOT add features outside the agreed scope
7. Each change must be runnable and verifiable — no half-built infrastructure

## Pending Decisions (DO NOT implement without resolution)

| Decision | Constraint |
|---|---|
| File preview approach | Do not add conversion services (LibreOffice, ffmpeg, etc.) until resolved |
| Infrastructure (CI/CD, deployment) | Only basic setup; no production infra |
| Folder permission inheritance model | Must be explicitly designed before implementing folder-level permissions |

## Completion Checklist

After any change, report:
- Files changed and what changed
- Why it changed
- What verification was run
- Any checks that were skipped (and why)
- Risks or follow-ups identified
