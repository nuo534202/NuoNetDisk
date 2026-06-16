# NuoNetDisk

A cloud storage application with a Go/Gin backend and React/TypeScript frontend.

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, Gin framework |
| Frontend | React, TypeScript, Vite |
| Metadata DB | PostgreSQL |
| File Storage | MinIO (S3-compatible) |
| Auth | JWT (access + refresh token rotation) |

## Prerequisites

- Docker & Docker Compose

## Quick Start

```bash
# Copy environment config
cp .env.example .env

# Start everything — PostgreSQL, MinIO, backend, and frontend
docker compose up -d
```

Open `http://localhost:3000` in your browser. The first build takes a minute (frontend compiles inside Docker).

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |

## Manual Setup (for development)

Run only the infrastructure in Docker, then run the backend and frontend natively for hot reload.

### 1. Infrastructure

```bash
docker compose up -d postgres minio
```

### 2. Configuration

```bash
cp .env.example .env
```

Key settings:

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://nuonetdisk:nuonetdisk_dev@localhost:5432/nuonetdisk?sslmode=disable` | PostgreSQL connection |
| `MINIO_ENDPOINT` | `localhost:9000` | MinIO server address |
| `MINIO_ACCESS_KEY` | `nuonetdisk` | MinIO access key |
| `MINIO_SECRET_KEY` | `nuonetdisk_dev` | MinIO secret key |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `SERVER_PORT` | `8080` | Backend HTTP port |
| `FRONTEND_PORT` | `3000` | Frontend container host port |

### 3. Backend

```bash
cd backend
go mod download
go run ./cmd/server/
```

Migrations run automatically on startup. Backend listens on `http://localhost:8080`.

### 4. Frontend

```bash
cd frontend
npm install
npm run dev
```

Dev server starts on `http://localhost:5173`. The frontend sends API requests to `http://localhost:8080/api/v1` (configure `VITE_API_BASE_URL` to change the target).

## Project Structure

```
nuonetdisk/
├── api/
│   └── openapi.yaml            # API specification (OpenAPI 3.0)
├── backend/
│   ├── cmd/server/main.go      # Application entry point
│   ├── internal/
│   │   ├── auth/               # JWT, bcrypt, refresh tokens
│   │   ├── config/             # Environment-based configuration
│   │   ├── handler/            # HTTP handlers
│   │   ├── middleware/         # Auth, CORS, rate limiter, logger, recovery
│   │   ├── model/              # Domain models and error types
│   │   ├── repository/         # PostgreSQL data access layer
│   │   ├── service/            # Business logic layer
│   │   └── storage/            # MinIO file storage client
│   ├── migrations/             # SQL migration files
│   └── pkg/httputil/           # HTTP utilities (responses, pagination)
├── frontend/
│   ├── src/
│   │   ├── pages/              # Route page components
│   │   ├── services/           # API client services
│   │   ├── store/              # Auth context
│   │   └── types/              # TypeScript type definitions
│   ├── Dockerfile              # Multi-stage frontend build
│   ├── nginx.conf              # Nginx config with API proxy
│   └── vite.config.ts
├── docker-compose.yml          # PostgreSQL + MinIO + backend + frontend
├── .env.example                # Environment config template
├── AGENTS.md                   # Agent instructions
└── README.md
```

## API Endpoints

All endpoints are prefixed with `/api/v1`.

### Auth
| Method | Path | Description |
|---|---|---|
| POST | `/auth/register` | Create account |
| POST | `/auth/login` | Sign in |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Sign out |

### User
| Method | Path | Description |
|---|---|---|
| GET | `/user/me` | Get current user profile |
| PATCH | `/user/me` | Update display name |

### Files
| Method | Path | Description |
|---|---|---|
| GET | `/files` | List files (query: `folder_id`, `offset`, `limit`) |
| POST | `/files` | Upload file (multipart form) |
| GET | `/files/:fileId` | Get file metadata |
| GET | `/files/:fileId/download` | Download file content |
| PATCH | `/files/:fileId` | Rename or move file |
| DELETE | `/files/:fileId` | Soft-delete file |

### Folders
| Method | Path | Description |
|---|---|---|
| POST | `/folders` | Create folder |
| GET | `/folders` | List folders (query: `parent_id`) |
| GET | `/folders/:folderId` | Get folder metadata |
| PATCH | `/folders/:folderId` | Rename or move folder |
| DELETE | `/folders/:folderId` | Soft-delete folder |

### Shares
| Method | Path | Description |
|---|---|---|
| POST | `/shares` | Create share link |
| DELETE | `/shares/:shareId` | Revoke share link |
| GET | `/shares/token/:token` | Access shared resource |

### Recycle Bin
| Method | Path | Description |
|---|---|---|
| GET | `/recycle-bin/files` | List deleted files |
| GET | `/recycle-bin/folders` | List deleted folders |
| POST | `/recycle-bin/restore/file/:fileId` | Restore file |
| POST | `/recycle-bin/restore/folder/:folderId` | Restore folder |
| DELETE | `/recycle-bin/file/:fileId` | Permanently delete file |
| DELETE | `/recycle-bin/folder/:folderId` | Permanently delete folder |

## Dev Scripts

### Backend

```bash
go build ./...           # Compile
go vet ./...             # Static analysis
go test ./...            # Run tests
gofmt -d .               # Check formatting
```

### Frontend

```bash
npm run dev              # Start dev server
npm run build            # Type-check and build
npm run typecheck        # TypeScript check only
npm run lint             # ESLint
npm run test             # Run tests
```
