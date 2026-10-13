# NuoNetDisk

NuoNetDisk is a self-hosted cloud storage platform. Upload, organize, and share your files through your browser — no third-party cloud provider required.

## Features

- **File management** — Upload, download, rename, and organize files in a folder hierarchy.
- **Folder navigation** — Create folders, drill into subdirectories, and navigate with breadcrumb trails.
- **Share via links** — Generate share links for files or folders with read/write permissions and optional expiration. Revoke anytime.
- **Recycle bin** — Deleted files and folders are soft-deleted with automatic expiration. Restore them before they're permanently removed.
- **Account management** — Register with email and password, update your display name, and manage your session.
- **JWT authentication** — Login state persists with access + refresh token rotation. Session ends on sign out.
- **End-to-end self-hosted** — Your data stays on your infrastructure. No data leaves your PostgreSQL and MinIO instances.

## Quick Start

Requires Docker and Docker Compose.

```bash
# 1. Clone the repository
git clone git@github.com:nuo534202/NuoNetDisk.git
cd nuonetdisk

# 2. Copy environment config
cp .env.example .env

# 3. Start the application
docker compose up -d
```

Open **http://localhost:3000** in your browser. The first startup may take a minute while the frontend compiles.

### What runs inside Docker

| Service | Access | Purpose |
|---|---|---|
| Frontend & Admin Panel | http://localhost:3000 | Web interface, Admin Panel |
| Backend API | http://localhost:8080 | REST API |
| MinIO Console | http://localhost:9001 | Object storage admin |

## Admin Panel

Access the admin panel at **http://localhost:3000** after signing in with an admin account.

### Default Admin Account

An admin account is automatically created on first startup and cannot be modified. Use these credentials to sign in:

| Setting | Default (dev) | Note |
|---|---|---|
| Admin email | `admin@example.com` | Set via `ADMIN_EMAIL` in `.env` |
| Admin password | `admin123` | Set via `ADMIN_PASSWORD` in `.env` |

After signing in at http://localhost:3000/login, click **Admin Panel** in the top navigation bar to access the dashboard.

The admin panel provides:
- **Project info** — Application name, version, uptime, Go runtime version.
- **Service health** — Live status indicators for PostgreSQL, MinIO, and the backend server.
- **System resources** — Goroutine count, memory allocation, GC cycles.
- **Storage statistics** — Total users, files, folders, storage usage, active shares, recycle bin count.
- **User management** — View all registered users, promote or revoke admin roles.

## Getting started

1. Open http://localhost:3000 and click **Create one** to register a new account.
2. Enter your email and password (at least 8 characters).
3. Sign in and upload your first file via the **Upload file** button.
4. Create folders to stay organized — use the **+ New folder** button.
5. Download files with the download button next to each file entry.
6. Deleted files go to **Recycle bin** (accessible from the top bar) where you can restore or permanently delete them.
7. Share files with others by creating share links (API feature — see API documentation for details).

## Default configuration

Default credentials and ports are set in `.env.example`. Change these before any production use:

| Setting | Default (dev) | Note |
|---|---|---|
| JWT secret | `change-me-in-production` | **Must change before production** |
| PostgreSQL password | `nuonetdisk_dev` | Change for production |
| MinIO password | `nuonetdisk_dev` | Change for production |
| Max upload size | 100 MB | Configurable via `MAX_UPLOAD_SIZE` |
| Recycle bin expiry | 30 days | Configurable via `RECYCLE_BIN_EXPIRY` |

## Security

- Passwords are hashed with bcrypt.
- File contents are stored in MinIO, never in the database.
- All file operations require authentication and authorization.
- File metadata (names, sizes) is stored in PostgreSQL; actual file objects are opaque keys in MinIO.
- Uploads are streamed directly to storage — no temporary files on disk.
- Share links are revocable, expirable, and scoped by permission.
- Download URLs are served through the backend — MinIO object keys are never exposed to the frontend.

## Architecture overview

NuoNetDisk uses three storage layers that work together:

- **PostgreSQL** — Stores metadata: users, files, folders, shares, recycle bin state.
- **MinIO** — Stores the actual file objects (S3-compatible API).
- **Backend (Go/Gin)** — REST API that mediates between the frontend, database, and object storage.

The frontend is a single-page application built with React and TypeScript (Vite).

## Development

See [AGENTS.md](./AGENTS.md) for project conventions and [api/openapi.yaml](./api/openapi.yaml) for the full API contract.

## License

See [LICENSE](./LICENSE) for details.