# Changelog

## v0.1.0 (2026-06-17)

### Features

- User registration, login, JWT access/refresh token rotation, and logout
- File upload with size limit enforcement, streaming download, rename, move, and soft-delete
- Folder create, list, rename, move, and soft-delete with cascade
- Share link creation, revocation, and token-based access
- Recycle bin with soft-delete, restore, permanent delete, and scheduled cleanup
- Rate limiting (per-IP, per-user, per-auth-endpoint)
- CORS configuration, request logging, panic recovery middleware
- Graceful server shutdown with context propagation
- Automated database migrations on startup

### Frontend

- React + TypeScript + Vite with React Router
- Dashboard with file/folder listing, upload, download, rename, move, delete
- Login and registration pages
- Profile page for display name updates
- Recycle bin page for restore and permanent delete
- Shared files page for accessing shared resources
- Auth state management with automatic token refresh

### Infrastructure

- Docker Compose setup for PostgreSQL, MinIO, backend, and frontend
- Production Dockerfiles with multi-stage builds
- Nginx config with API reverse proxy
- Environment-based configuration

### Security

- Passwords hashed with bcrypt (cost 10)
- JWT access tokens with configurable expiry
- Refresh token rotation with revocation
- File size limits enforced before reading body
- Internal UUID-based object keys (no user-supplied filenames in MinIO)
- Server-side MIME type detection
- Path traversal prevention
- Per-operation authorization checks

### Notes

- Initial MVP release
- Zero test coverage in previous state — test infrastructure added in this release
- File preview not yet implemented
- Folder permission inheritance model not yet implemented
- CI/CD pipeline not yet configured
