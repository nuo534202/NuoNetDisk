# NuoNetDisk — Architecture Design Document

> Version 1.0
> Status: Approved
> Based on: AGENTS.md, docs/requirements.md

---

## Table of Contents

- [NuoNetDisk — Architecture Design Document](#nuonetdisk--architecture-design-document)
  - [Table of Contents](#table-of-contents)
  - [1. System Overview](#1-system-overview)
    - [1.1 Architecture Philosophy](#11-architecture-philosophy)
    - [1.2 High-Level Architecture Diagram](#12-high-level-architecture-diagram)
    - [1.3 Technology Stack Summary](#13-technology-stack-summary)
  - [2. Backend Layered Architecture](#2-backend-layered-architecture)
    - [2.1 Layer Overview](#21-layer-overview)
    - [2.2 Layer Dependency Rules](#22-layer-dependency-rules)
    - [2.3 Package Structure](#23-package-structure)
    - [2.4 Module Responsibility Matrix](#24-module-responsibility-matrix)
  - [3. Handler Layer](#3-handler-layer)
    - [3.1 Responsibility](#31-responsibility)
    - [3.2 Handler Module Design](#32-handler-module-design)
    - [3.3 Request/Response Conventions](#33-requestresponse-conventions)
    - [3.4 Key Interface](#34-key-interface)
  - [4. Service Layer](#4-service-layer)
    - [4.1 Responsibility](#41-responsibility)
    - [4.2 Service Module Design](#42-service-module-design)
    - [4.3 Key Interfaces](#43-key-interfaces)
    - [4.4 Orchestration Rules](#44-orchestration-rules)
  - [5. Repository Layer](#5-repository-layer)
    - [5.1 Responsibility](#51-responsibility)
    - [5.2 Repository Module Design](#52-repository-module-design)
    - [5.3 Key Interfaces](#53-key-interfaces)
    - [5.4 Query Pattern](#54-query-pattern)
  - [6. Storage Layer](#6-storage-layer)
    - [6.1 Responsibility](#61-responsibility)
    - [6.2 Storage Module Design](#62-storage-module-design)
    - [6.3 Key Interface](#63-key-interface)
    - [6.4 Partial Failure Handling](#64-partial-failure-handling)
  - [7. Auth Module](#7-auth-module)
    - [7.1 Architecture](#71-architecture)
    - [7.2 JWT Token Design](#72-jwt-token-design)
    - [7.3 Refresh Token Rotation Protocol](#73-refresh-token-rotation-protocol)
    - [7.4 Auth Middleware](#74-auth-middleware)
  - [8. Middleware Chain](#8-middleware-chain)
    - [8.1 Chain Order](#81-chain-order)
    - [8.2 Middleware Specifications](#82-middleware-specifications)
    - [8.3 Public Endpoint Whitelist](#83-public-endpoint-whitelist)
  - [9. Database Schema Design](#9-database-schema-design)
    - [9.1 Entity-Relationship Diagram](#91-entity-relationship-diagram)
    - [9.2 Table Definitions](#92-table-definitions)
      - [`users`](#users)
      - [`folders`](#folders)
      - [`files`](#files)
      - [`share_links`](#share_links)
      - [`refresh_tokens`](#refresh_tokens)
      - [Permission model placeholder](#permission-model-placeholder)
    - [9.3 Indexing Strategy](#93-indexing-strategy)
    - [9.4 Migration Strategy](#94-migration-strategy)
  - [10. API Design](#10-api-design)
    - [10.1 Design Conventions](#101-design-conventions)
    - [10.2 Error Response Format](#102-error-response-format)
    - [10.3 Pagination](#103-pagination)
    - [10.4 Core Endpoint Specifications](#104-core-endpoint-specifications)
      - [Authentication](#authentication)
      - [Files](#files-1)
      - [Folders](#folders-1)
      - [Share Links](#share-links)
      - [Recycle Bin](#recycle-bin)
      - [User](#user)
  - [11. Data Flow for Critical Operations](#11-data-flow-for-critical-operations)
    - [11.1 File Upload Flow](#111-file-upload-flow)
    - [11.2 File Download Flow](#112-file-download-flow)
    - [11.3 Soft-Delete + Recycle Bin Flow](#113-soft-delete--recycle-bin-flow)
    - [11.4 Folder Delete Cascade Flow](#114-folder-delete-cascade-flow)
    - [11.5 Share Link Access Flow](#115-share-link-access-flow)
    - [11.6 File Rename with Optimistic Locking](#116-file-rename-with-optimistic-locking)
  - [12. Error Handling Strategy](#12-error-handling-strategy)
    - [12.1 Error Types](#121-error-types)
    - [12.2 Error Propagation](#122-error-propagation)
    - [12.3 Global Error Handler](#123-global-error-handler)
  - [13. Scheduled Tasks](#13-scheduled-tasks)
    - [13.1 Recycle Bin Cleanup](#131-recycle-bin-cleanup)
    - [13.2 Architecture](#132-architecture)
  - [14. Frontend Architecture](#14-frontend-architecture)
    - [14.1 Directory Structure](#141-directory-structure)
    - [14.2 Component Hierarchy](#142-component-hierarchy)
    - [14.3 State Management](#143-state-management)
    - [14.4 API Client Layer](#144-api-client-layer)
    - [14.5 Route Design](#145-route-design)
  - [15. Local Development Architecture](#15-local-development-architecture)
    - [15.1 Development Environment (Docker Compose)](#151-development-environment-docker-compose)
    - [15.2 Service Configuration](#152-service-configuration)
    - [15.3 Environment Variables](#153-environment-variables)
    - [15.4 Network Architecture (Development)](#154-network-architecture-development)
  - [16. Security Architecture](#16-security-architecture)
    - [16.1 Defense in Depth](#161-defense-in-depth)
    - [16.2 Threat Model Summary](#162-threat-model-summary)
  - [17. Pending Decisions \& Future Considerations](#17-pending-decisions--future-considerations)
  - [Appendix A: Go Dependency Justification](#appendix-a-go-dependency-justification)
  - [Appendix B: Key Go Interface Summary](#appendix-b-key-go-interface-summary)

---

## 1. System Overview

### 1.1 Architecture Philosophy

NuoNetDisk follows a **backend monolith with layered architecture**:

| Principle | Application |
|---|---|
| **Separation of Concerns** | Handler → Service → Repository + Storage — each layer has one job |
| **Interface-based Programming** | Go interfaces at layer boundaries for testability and loose coupling |
| **Streaming by Default** | No file content loaded entirely into memory in any code path |
| **Defense in Depth** | Auth at middleware layer + authorization at service layer + input validation at handler layer |
| **Explicit Partial Failure** | Every dual-write (MinIO + PostgreSQL) explicitly handles the case where one succeeds and the other fails |
| **API-First** | OpenAPI contract defined before handler implementation |
| **Incremental Structure** | Directories created as features grow, not pre-created |

### 1.2 High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Browser (SPA)                               │
│                   React + TypeScript + Vite                          │
│                                                                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────────┐   │
│  │   Pages  │ │Components│ │   Hooks  │ │  Services (API client)│   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────────┘   │
└────────────────────────────┬────────────────────────────────────────┘
                             │ HTTPS / HTTP
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Gin HTTP Server (Go)                             │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Middleware Chain                           │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐ │    │
│  │  │ Recovery │ │ Logging  │ │   CORS   │ │   Rate Limit   │ │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └────────────────┘ │    │
│  │  ┌────────────────┐ ┌──────────────────┐ ┌──────────────┐  │    │
│  │  │ Auth (JWT)     │ │ Request Context  │ │  ...         │  │    │
│  │  └────────────────┘ └──────────────────┘ └──────────────┘  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   Handler Layer (HTTP)                       │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐ │    │
│  │  │   Auth   │ │  Files   │ │ Folders  │ │   Shares       │ │    │
│  │  │ Handlers │ │ Handlers │ │ Handlers │ │   Handlers     │ │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └────────────────┘ │    │
│  │  ┌──────────────┐ ┌──────────────┐                         │    │
│  │  │ Recycle Bin  │ │   User       │                         │    │
│  │  │  Handlers    │ │  Handlers    │                         │    │
│  │  └──────────────┘ └──────────────┘                         │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                       │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   Service Layer (Business Logic)             │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐ │    │
│  │  │AuthService│ │FileService│ │FolderSvc │ │  ShareService  │ │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └────────────────┘ │    │
│  │  ┌──────────────┐ ┌──────────────┐                         │    │
│  │  │ RecycleSvc   │ │  UserSvc     │                         │    │
│  │  └──────────────┘ └──────────────┘                         │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                       │
│              ┌───────────────┴───────────────┐                       │
│              ▼                               ▼                       │
│  ┌──────────────────────┐   ┌──────────────────────────────┐        │
│  │  Repository Layer    │   │   Storage Layer               │        │
│  │  (PostgreSQL Access) │   │   (MinIO Abstraction)         │        │
│  │                      │   │                              │        │
│  │ ┌──────────────────┐ │   │ ┌──────────────────────────┐ │        │
│  │ │   UserRepo       │ │   │ │   FileStorage (interface)│ │        │
│  │ │   FileRepo       │ │   │ │   - Upload(ctx, ...)     │ │        │
│  │ │   FolderRepo     │ │   │ │   - Download(ctx, ...)   │ │        │
│  │ │   ShareRepo      │ │   │ │   - Delete(ctx, ...)     │ │        │
│  │ │   RefreshTokenRepo│ │   │ │   - Copy(ctx, ...)      │ │        │
│  │ └──────────────────┘ │   │ └──────────────────────────┘ │        │
│  └──────────────────────┘   └──────────────────────────────┘        │
└─────────────┬──────────────────────────────┬────────────────────────┘
              │                              │
              │        TCP/5432              │        TCP/9000
              ▼                              ▼
┌─────────────────────────┐   ┌──────────────────────────────┐
│      PostgreSQL          │   │       MinIO (S3 API)          │
│   (Metadata Database)    │   │    (File Object Storage)      │
│                          │   │                              │
│  Tables:                 │   │  Bucket: "nuonetdisk"        │
│  - users                 │   │                              │
│  - files                 │   │  Object Keys:                │
│  - folders               │   │  UUID-based, opaque          │
│  - share_links           │   │  e.g., "d4e7f1a2-..."       │
│  - refresh_tokens        │   │                              │
│  - schema_migrations     │   │  No user-identifiable info   │
└─────────────────────────┘   └──────────────────────────────┘
```

### 1.3 Technology Stack Summary

| Layer | Technology | Purpose |
|---|---|---|
| Frontend Framework | React + TypeScript | Fixed frontend stack for SPA UI |
| Build Tool | Vite | Fixed frontend build tool |
| HTTP Client | Native `fetch` by default | Add axios only with explicit dependency approval |
| State Management | React Context + useReducer | Client-side state without extra runtime dependency |
| Styling | Plain CSS / CSS Modules by default | Add Tailwind CSS only with explicit dependency approval |
| Backend Language | Go | Fixed backend language |
| HTTP Framework | Gin | Fixed backend framework; module addition still requires approval |
| DB Driver | `database/sql` + PostgreSQL driver or pgx | Driver choice requires explicit dependency approval |
| Migrations | Plain SQL migration files | Migration runner/tool requires explicit dependency approval |
| Object Storage | MinIO S3 API | SDK choice requires explicit dependency approval |
| Auth | JWT | JWT implementation/library requires explicit dependency approval |
| Password Hashing | bcrypt or equivalent | Package choice requires explicit dependency approval |
| Configuration | Standard env vars by default | Add envconfig/viper only with explicit approval |
| Containerization | Docker + Docker Compose | Local development only |
| Reverse Proxy | Optional local-dev proxy/static server | Nginx is not required for MVP and needs approval if added |
| API Spec | OpenAPI 3.x (YAML) | API contract definition |

---

## 2. Backend Layered Architecture

### 2.1 Layer Overview

```
┌─────────────────────────────────────────┐
│            Handler Layer                 │  ← HTTP concerns (parse request, write response)
├─────────────────────────────────────────┤
│            Service Layer                 │  ← Business logic, orchestration, authorization
├─────────────────────────────────────────┤
│    ┌──────────────┐  ┌──────────────┐   │
│    │ Repository   │  │   Storage    │   │  ← Data access (DB queries, MinIO operations)
│    │ Layer        │  │   Layer      │   │
│    └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────┘
```

### 2.2 Layer Dependency Rules

| Rule | Description |
|---|---|
| **R1** | Handlers call Services only. Handlers never call Repositories or Storage directly. |
| **R2** | Services call Repositories and Storage through interfaces. Services never depend on Gin or HTTP concepts. |
| **R3** | Repositories depend on the approved PostgreSQL driver only. Repository implementations are never exposed to Handlers. |
| **R4** | Storage implementation depends only on the approved MinIO/S3 client. Storage interface is defined in the service layer (where it's needed). |
| **R5** | Dependency injection via constructor functions (e.g., `NewFileService(repo FileRepository, storage FileStorage)`) — no global state or service locator. |

```
Handler ──→ Service ──→ Repository (interface)
                        └──→ PostgreSQL
              │
              └──→ Storage (interface)
                    └──→ MinIO
```

### 2.3 Package Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: config loading, DI wiring, server start
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration struct + loading from env vars
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication middleware
│   │   ├── cors.go              # CORS middleware
│   │   ├── ratelimit.go         # Rate limiting middleware
│   │   ├── logger.go            # Request logging with redaction
│   │   └── recovery.go          # Panic recovery
│   ├── handler/
│   │   ├── auth.go              # Register, login, refresh, logout
│   │   ├── file.go              # File CRUD endpoints
│   │   ├── folder.go            # Folder CRUD endpoints
│   │   ├── share.go             # Share link CRUD + access
│   │   ├── recycle_bin.go       # List, restore, permanent delete
│   │   └── user.go              # Get current user profile
│   ├── service/
│   │   ├── auth.go              # Auth business logic
│   │   ├── file.go              # File business logic
│   │   ├── folder.go            # Folder business logic
│   │   ├── share.go             # Share link business logic
│   │   ├── recycle_bin.go       # Recycle bin business logic
│   │   └── user.go              # User business logic
│   ├── repository/
│   │   ├── user.go              # User DB operations
│   │   ├── file.go              # File metadata DB operations
│   │   ├── folder.go            # Folder metadata DB operations
│   │   ├── share.go             # Share link DB operations
│   │   └── refresh_token.go     # Refresh token DB operations
│   ├── model/
│   │   ├── user.go              # User domain model
│   │   ├── file.go              # File domain model
│   │   ├── folder.go            # Folder domain model
│   │   ├── share.go             # Share link domain model
│   │   └── errors.go            # Domain error types
│   ├── auth/
│   │   ├── jwt.go               # JWT generation & validation
│   │   ├── refresh_token.go     # Refresh token generation & hashing
│   │   └── password.go          # Password hashing & verification
│   └── storage/
│       └── minio.go             # MinIO client abstraction (implements service.Storage interface)
├── pkg/
│   └── httputil/
│       └── response.go          # JSON response helpers + error response
├── migrations/
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_folders.up.sql
│   ├── 000002_create_folders.down.sql
│   ├── 000003_create_files.up.sql
│   ├── 000003_create_files.down.sql
│   ├── 000004_create_share_links.up.sql
│   ├── 000004_create_share_links.down.sql
│   ├── 000005_create_refresh_tokens.up.sql
│   └── 000005_create_refresh_tokens.down.sql
├── go.mod
└── go.sum
```

> **Incremental note**: This is the target structure. Create directories and files incrementally — for example, `handler/`, `service/`, `repository/` directories appear when the first feature (auth) is implemented. See AGENTS.md.

### 2.4 Module Responsibility Matrix

| Package | Responsibility | Dependencies | Avoid |
|---|---|---|---|
| `config` | Load + validate all config from env vars | None | Hardcoded values, secrets in code |
| `middleware` | Gin middleware handlers | `auth` (for auth MW) | Business logic, DB access |
| `handler` | HTTP request parsing, response writing, validation | `service`, `auth` | DB queries, MinIO calls, business logic |
| `service` | Business logic, authorization, orchestration | `repository`, `storage`, `auth`, `model` | HTTP concepts (gin.Context, etc.) |
| `repository` | SQL queries, DB access | `model`, database driver | Business logic, HTTP, MinIO |
| `storage` | MinIO operations | Approved MinIO/S3 client SDK | Business logic, DB access |
| `auth` | JWT, bcrypt, refresh token crypto | None (utility package) | HTTP, DB (called by services) |
| `model` | Domain types, error types | None | Any infrastructure dependency |
| `migrations` | SQL migration files | Selected migration runner only after explicit approval | --- |
| `cmd/server` | DI wiring, server bootstrap | All internal packages | Business logic |

---

## 3. Handler Layer

### 3.1 Responsibility

- Parse and validate HTTP requests (path params, query params, JSON body, multipart form)
- Call the corresponding service method
- Format and write HTTP responses (JSON for data, streaming for download)
- Never contain business logic or database queries
- Handle HTTP-specific errors (400 bad request, 401 unauthorized, 404 not found, 500 internal)

### 3.2 Handler Module Design

Each handler file is a single file (not a package) within `internal/handler/`, grouped by resource:

```go
// internal/handler/file.go

type FileHandler struct {
    fileService service.FileService
}

func NewFileHandler(fileService service.FileService) *FileHandler {
    return &FileHandler{fileService: fileService}
}

func (h *FileHandler) List(c *gin.Context)    { ... }
func (h *FileHandler) Upload(c *gin.Context)  { ... }
func (h *FileHandler) Get(c *gin.Context)     { ... }
func (h *FileHandler) Download(c *gin.Context) { ... }
func (h *FileHandler) Update(c *gin.Context)  { ... }   // rename/move
func (h *FileHandler) Delete(c *gin.Context)  { ... }   // soft-delete
```

### 3.3 Request/Response Conventions

**Request parsing pattern:**
```go
func (h *FileHandler) Update(c *gin.Context) {
    var req UpdateFileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
        return
    }
    // Call service
    userID := middleware.GetUserID(c)
    file, err := h.fileService.Update(c.Request.Context(), userID, req)
    if err != nil {
        httputil.RespondServiceError(c, err)
        return
    }
    httputil.RespondJSON(c, http.StatusOK, file)
}
```

**Response helpers (`pkg/httputil/response.go`):**
```go
func RespondJSON(c *gin.Context, status int, data any)
func RespondError(c *gin.Context, status int, code string, message string)
func RespondServiceError(c *gin.Context, err error)  // maps domain errors to HTTP
func RespondStream(c *gin.Context, status int, contentType string, reader io.Reader)
```

**Error response format:**
```json
{
    "error": {
        "code": "FILE_NOT_FOUND",
        "message": "File not found or access denied"
    }
}
```

### 3.4 Key Interface

Handlers depend on service interfaces — the service package defines the interface, the handler consumes it:

```go
// service/file.go — interface defined by the service layer
type FileService interface {
    List(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, offset, limit int) ([]model.File, int, error)
    Upload(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, filename string, size int64, reader io.Reader) (*model.File, error)
    Get(ctx context.Context, userID uuid.UUID, fileID uuid.UUID) (*model.File, error)
    Download(ctx context.Context, userID uuid.UUID, fileID uuid.UUID) (*model.File, io.ReadCloser, error)
    Update(ctx context.Context, userID uuid.UUID, fileID uuid.UUID, req UpdateFileRequest) (*model.File, error)
    Delete(ctx context.Context, userID uuid.UUID, fileID uuid.UUID) error
}
```

---

## 4. Service Layer

### 4.1 Responsibility

- Implement business logic and workflow orchestration
- Enforce authorization (resource ownership / share-link permission)
- Coordinate between Repository (DB) and Storage (MinIO) layers
- Handle partial failure in dual-write operations
- Define interfaces consumed by handlers, and interfaces it needs from repositories/storage
- Never reference HTTP concepts (`gin.Context`, `http.Request`, etc.)

### 4.2 Service Module Design

Each service is a struct that implements an interface used by handlers:

```go
// service/file.go
type fileService struct {
    repo    FileRepository      // interface
    storage FileStorage         // interface
    authSvc AuthService         // for permission checks
}

func NewFileService(repo FileRepository, storage FileStorage, authSvc AuthService) FileService {
    return &fileService{repo: repo, storage: storage, authSvc: authSvc}
}
```

### 4.3 Key Interfaces

Interfaces are defined **in the service package** (the consumer), not in the implementation package (the producer). This follows Go's convention: "accept interfaces, return structs."

```go
// service/file.go — repository interface (implemented by repository package)
type FileRepository interface {
    Insert(ctx context.Context, f *model.File) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.File, error)
    ListByFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, offset, limit int) ([]model.File, int, error)
    Update(ctx context.Context, f *model.File) error
    SoftDelete(ctx context.Context, id uuid.UUID) error
    // ... additional methods
}

// service/file.go — storage interface (implemented by storage package)
type FileStorage interface {
    Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
    Download(ctx context.Context, objectKey string) (io.ReadCloser, error)
    Delete(ctx context.Context, objectKey string) error
    Copy(ctx context.Context, srcKey, dstKey string) error
}
```

### 4.4 Orchestration Rules

```
Service method orchestration pattern:
  1. Validate inputs (format, permissions)
  2. Read current state from Repository
  3. Check authorization (is user the owner?)
  4. Perform primary operation (MinIO: upload/download/delete)
  5. Perform secondary operation (DB: insert/update/delete)
  6. On partial failure: clean up primary, return error
  7. Return result
```

**Upload orchestration (partial failure handling):**
```go
func (s *fileService) Upload(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, filename string, size int64, reader io.Reader) (*model.File, error) {
    // 1. Generate opaque object key (UUID-based)
    objectKey := uuid.New().String()

    // 2. Upload to MinIO first
    if err := s.storage.Upload(ctx, objectKey, reader, size, mimeType); err != nil {
        return nil, fmt.Errorf("storage upload: %w", err)
    }

    // 3. Insert metadata into DB
    file := &model.File{
        ID:        uuid.New(),
        UserID:    userID,
        ParentFolderID: folderID,
        Name:      sanitizeFilename(filename),
        ObjectKey: objectKey,
        Size:      size,
        MimeType:  mimeType,
        SHA256:    hash,
        Version:   1,
    }

    if err := s.repo.Insert(ctx, file); err != nil {
        // 4. MinIO succeeded but DB failed — clean up MinIO object
        cleanupErr := s.storage.Delete(ctx, objectKey)
        if cleanupErr != nil {
            log.Printf("CRITICAL: MinIO cleanup failed for key %s: %v", objectKey, cleanupErr)
            // Log and alert; don't leave dangling objects silently
        }
        return nil, fmt.Errorf("db insert: %w", err)
    }

    return file, nil
}
```

---

## 5. Repository Layer

### 5.1 Responsibility

- Execute SQL queries against PostgreSQL
- Map database rows to domain models (`model.*`)
- Implement the interfaces defined by the service layer
- Use parameterized queries exclusively (no string concatenation for SQL)
- Handle transaction management (for cascade operations)

### 5.2 Repository Module Design

```go
// repository/file.go
type fileRepository struct {
    db *sql.DB  // or pgxpool.Pool
}

func NewFileRepository(db *sql.DB) service.FileRepository {
    return &fileRepository{db: db}
}

func (r *fileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
    query := `SELECT id, user_id, name, object_key, size, mime_type, sha256_hash,
                     parent_folder_id, version, created_at, updated_at
              FROM files WHERE id = $1 AND is_deleted = false`
    row := r.db.QueryRowContext(ctx, query, id)
    // scan into model.File...
}
```

### 5.3 Key Interfaces

Defined in the `service` package (see §4.3).

### 5.4 Query Pattern

| Pattern | Usage |
|---|---|
| `QueryRowContext` / `QueryContext` | Standard queries |
| `ExecContext` | INSERT, UPDATE, DELETE |
| `BeginTx` | Transactions (folder delete cascade, restore) |
| `CopyFrom` (pgx, only if pgx is approved) | Bulk inserts (not expected for MVP) |

**Transaction usage (folder delete cascade):**
```go
func (r *fileRepository) SoftDeleteCascade(ctx context.Context, folderID uuid.UUID) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback() // no-op if committed

    // Set deleted_at on the folder
    _, err = tx.ExecContext(ctx, `UPDATE folders SET is_deleted=true, deleted_at=NOW() WHERE id = $1`, folderID)
    if err != nil {
        return err
    }

    // Cascade to all files in subtree
    _, err = tx.ExecContext(ctx, `
        UPDATE files SET is_deleted=true, deleted_at=NOW()
        WHERE parent_folder_id IN (
            WITH RECURSIVE subfolders AS (
                SELECT id FROM folders WHERE id = $1
                UNION ALL
                SELECT f.id FROM folders f JOIN subfolders s ON f.parent_folder_id = s.id
            )
            SELECT id FROM subfolders
        )`, folderID)
    if err != nil {
        return err
    }

    // Cascade to all sub-folders
    _, err = tx.ExecContext(ctx, `
        UPDATE folders SET is_deleted=true, deleted_at=NOW()
        WHERE id IN (
            WITH RECURSIVE subfolders AS (
                SELECT id FROM folders WHERE id = $1
                UNION ALL
                SELECT f.id FROM folders f JOIN subfolders s ON f.parent_folder_id = s.id
            )
            SELECT id FROM subfolders WHERE id != $1
        )`, folderID)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

---

## 6. Storage Layer

### 6.1 Responsibility

- Abstract MinIO S3 operations behind a Go interface
- Handle object upload/download/delete/copy
- Generate pre-signed URLs only if explicitly approved later
- Never store metadata — MinIO is only for raw file objects
- Retry only safe/idempotent operations or SDK-managed multipart uploads; never buffer full file content in memory for retry

### 6.2 Storage Module Design

```go
// storage/minio.go
type minioStorage struct {
    client   *minio.Client
    bucket   string
}

func NewMinioStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*minioStorage, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        return nil, err
    }
    // Ensure bucket exists
    return &minioStorage{client: client, bucket: bucket}, nil
}

func (s *minioStorage) Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
    _, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size,
        minio.PutObjectOptions{ContentType: contentType})
    return err
}

func (s *minioStorage) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
    obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
    if err != nil {
        return nil, err
    }
    return obj, nil
}

func (s *minioStorage) Delete(ctx context.Context, objectKey string) error {
    return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *minioStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
    _, err := s.client.CopyObject(ctx, s.bucket, dstKey, s.bucket, srcKey, minio.CopySrcOptions{}, minio.PutObjectOptions{})
    return err
}
```

### 6.3 Key Interface

Defined in service package — consumed by services, implemented by storage package:

```go
// service/file.go (or a separate storage interface file)
type FileStorage interface {
    Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
    Download(ctx context.Context, objectKey string) (io.ReadCloser, error)
    Delete(ctx context.Context, objectKey string) error
    Copy(ctx context.Context, srcKey, dstKey string) error
}
```

### 6.4 Partial Failure Handling

The storage layer itself does not handle partial failure — that's the service layer's responsibility. But the storage layer MUST:

1. Return meaningful errors that the service layer can act on (wrap with context, no secrets in error messages)
2. Support idempotent `Delete` (calling Delete on a non-existent key should succeed)
3. Retry transient S3 errors only where retry is safe:
   - `Delete` can be retried because it is idempotent.
   - `Download`/`Copy` may retry before a response body is committed.
   - `Upload` MUST NOT wrap a non-seekable request `io.Reader` in app-level retries, because retrying would require rereading the stream. Rely on the MinIO SDK's multipart behavior or require an explicitly seekable source; never use `bytes.Buffer` or any in-memory full-file buffer for retry.

---

## 7. Auth Module

### 7.1 Architecture

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────────────┐
│  AuthHandler  │────→│   AuthService    │────→│  RefreshTokenRepo    │
│  (HTTP)       │     │  (business logic)│     │  (DB operations)     │
└──────────────┘     └──────────────────┘     └──────────────────────┘
                           │
                           ▼
                     ┌──────────────┐
                     │   auth/jwt   │
                     │   (JWT ops)  │
                     └──────────────┘
```

### 7.2 JWT Token Design

| Property | Access Token | Refresh Token |
|---|---|---|
| **Format** | JWT (signed, not encrypted) | Opaque `token_id.secret` string |
| **Storage** | Client memory (not localStorage) | DB stores only bcrypt hash of secret; client stores opaque token in httpOnly cookie or memory |
| **Lifetime** | 15 minutes (configurable) | 7 days (configurable) |
| **Server-side state** | No (stateless) | Yes (look up token row by `token_id`, verify secret hash) |
| **Revocation** | Not directly (short TTL) | Set `is_revoked` in DB |
| **Rotation** | Not applicable | Yes — each use revokes old token and generates a new token |

**Access Token claims:**
```json
{
    "sub": "user-uuid",
    "email": "user@example.com",
    "iat": 1718000000,
    "exp": 1718000900,
    "type": "access"
}
```

**JWT signing:** Use HMAC-SHA256 (`HS256`) with a configurable secret key. In production, RSA or ECDSA (`RS256`/`ES256`) is preferred for asymmetric signing, but HS256 is simpler for a self-hosted single-deployment app.

### 7.3 Refresh Token Rotation Protocol

The rotation protocol is custom email/password authentication with refresh-token rotation and replay detection:

```
Client                    Server
  │                         │
  │  Login (email, pw)      │
  │────────────────────────→│
  │                         │  Verify password
  │                         │  Generate access_token (15m)
  │                         │  Generate refresh_token as token_id.secret
  │                         │  Store token_id + bcrypt(secret) in DB
  │  { access_token,        │
  │    refresh_token }      │
  │←────────────────────────│
  │                         │
  │  ...access_token expires │
  │                         │
  │  Refresh (refresh_token) │
  │────────────────────────→│
  │                         │  Parse token_id + secret
  │                         │  Look up token row by token_id
  │                         │  Verify bcrypt(secret) matches token_hash
  │                         │  If token is revoked → REVOKE ALL
  │                         │    user's refresh tokens (replay detection)
  │                         │  Revoke old token (set is_revoked=true)
  │                         │  Issue new access_token + new refresh_token
  │                         │  Store new token_id + bcrypt(secret) in DB
  │  { new_access_token,    │
  │    new_refresh_token }  │
  │←────────────────────────│
```

**Replay detection:** If a revoked refresh token is presented:
1. Mark all active refresh tokens for that user as revoked
2. Return 401 Unauthorized
3. This indicates the token may have been stolen — force re-login

```go
// auth/refresh_token.go
const SecretBytes = 32
const BcryptCost = 10

type RefreshTokenParts struct {
    ID     uuid.UUID
    Secret string
}

func GenerateRefreshToken() (token string, id uuid.UUID, secretHash string, err error) {
    id = uuid.New()
    secretBytes := make([]byte, SecretBytes)
    if _, err := rand.Read(secretBytes); err != nil {
        return "", uuid.Nil, "", err
    }
    secret := base64.RawURLEncoding.EncodeToString(secretBytes)
    hashBytes, err := bcrypt.GenerateFromPassword([]byte(secret), BcryptCost)
    if err != nil {
        return "", uuid.Nil, "", err
    }
    token = id.String() + "." + secret
    return token, id, string(hashBytes), nil
}

func VerifyRefreshTokenSecret(secret, hash string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}
```

### 7.4 Auth Middleware

```go
// middleware/auth.go
func JWTAuth(jwtService *auth.JWTService, publicPaths []string) gin.HandlerFunc {
    publicSet := make(map[string]bool, len(publicPaths))
    for _, p := range publicPaths {
        publicSet[p] = true
    }

    return func(c *gin.Context) {
        path := c.FullPath()
        if publicSet[path] {
            c.Next()
            return
        }

        tokenStr := extractBearerToken(c.GetHeader("Authorization"))
        if tokenStr == "" {
            c.AbortWithStatusJSON(401, errorResponse("MISSING_TOKEN", "Authorization header required"))
            return
        }

        claims, err := jwtService.ValidateAccessToken(tokenStr)
        if err != nil {
            c.AbortWithStatusJSON(401, errorResponse("INVALID_TOKEN", "Invalid or expired token"))
            return
        }

        c.Set("user_id", claims.Subject)
        c.Next()
    }
}
```

**Public endpoint whitelist:**
| Method | Path | Note |
|---|---|---|
| POST | /api/v1/auth/register | User registration |
| POST | /api/v1/auth/login | User login |
| POST | /api/v1/auth/refresh | Token refresh (validates refresh token, not JWT) |
| GET | /api/v1/shares/:token | Share link access (validates share token) |

---

## 8. Middleware Chain

### 8.1 Chain Order

Middleware order matters in Gin. The order below is applied to the **global router** (public endpoints bypass auth via path whitelist):

```
Request
  │
  ▼
1. Recovery              ──  Recover from panics, return 500
2. Logger (redacted)     ──  Log request method, status, duration, sanitized path
3. CORS                  ──  Set CORS headers
4. Rate Limiter          ──  Per-IP + per-user rate limiting
5. Auth (JWT)            ──  Extract/validate JWT, skip public paths
6. Request Context       ──  Inject request ID, set up context
  │
  ▼
Handler
```

### 8.2 Middleware Specifications

**1. Recovery Middleware** (Gin built-in `gin.Recovery()`):
- Catches panics, logs stack trace, returns 500 JSON error
- Does not expose stack traces in production

**2. Logging Middleware** (custom):
```go
// middleware/logger.go
func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path  // raw path before handler may modify it

        c.Next()

        duration := time.Since(start)
        status := c.Writer.Status()

        // Log sanitized fields only
        log.Printf("[%d] %s %s %s",
            status,
            c.Request.Method,
            sanitizePath(path),      // strip share tokens, query params with secrets
            duration,
        )
    }
}
```

**Sensitive data redaction rules:**
- Path segments matching `:token` (share link tokens) → replace with `[redacted]`
- Query string `?token=...` → replace value with `[redacted]`
- `Authorization` header → never logged
- Request body → never logged for auth endpoints

**3. CORS Middleware:**
```go
func CORS(allowedOrigins []string) gin.HandlerFunc {
    // In development: allow all origins
    // In production: restrict to frontend domain
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", determineOrigin(c, allowedOrigins))
        c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
        c.Header("Access-Control-Allow-Credentials", "true")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    }
}
```

**4. Rate Limiter (per-IP + per-user):**
```go
// middleware/ratelimit.go
type RateLimiter struct {
    ipLimiter    *rateLimiterStore   // per-IP: 100 req/s burst 200
    userLimiter  *rateLimiterStore   // per-user: 1000 req/min
    uploadLimiter *rateLimiterStore  // per-user upload: 10 concurrent
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        if !rl.ipLimiter.Allow(ip) {
            c.AbortWithStatusJSON(429, errorResponse("RATE_LIMITED", "Too many requests"))
            return
        }

        // Per-user limit (if authenticated)
        userID, exists := c.Get("user_id")
        if exists {
            if !rl.userLimiter.Allow(userID.(string)) {
                c.AbortWithStatusJSON(429, errorResponse("RATE_LIMITED", "Too many requests"))
                return
            }
        }
        c.Next()
    }
}
```

**5. Auth Middleware** (see §7.4)

### 8.3 Public Endpoint Whitelist

| Path | Method | Auth Required | Notes |
|---|---|---|---|
| /api/v1/auth/register | POST | No | |
| /api/v1/auth/login | POST | No | |
| /api/v1/auth/refresh | POST | No (uses refresh token) | |
| /api/v1/auth/logout | POST | Yes | |
| /api/v1/shares/:token | GET | No (uses share token) | Share link access |
| /api/v1/* | All | Yes | All other endpoints |

---

## 9. Database Schema Design

### 9.1 Entity-Relationship Diagram

```
┌──────────┐       ┌──────────┐       ┌───────────────┐
│  users   │1──N→│  files   │N──→1│   folders      │
└──────────┘       └──────────┘       └───────────────┘
     │1                  │1                  │1
     │                   │                   │
     │                   │                   │
     ▼                   ▼                   ▼
┌──────────────┐  ┌──────────────┐  ┌───────────────┐
│refresh_tokens│  │ share_links  │  │ (self-ref)    │
│  N──→1 users  │  │  N──→1 users │  │ parent_folder │
└──────────────┘  └──────────────┘  └───────────────┘
```

### 9.2 Table Definitions

#### `users`

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
```

#### `folders`

```sql
CREATE TABLE folders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    parent_folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_folder_parent FOREIGN KEY (parent_folder_id) REFERENCES folders(id)
);

CREATE INDEX idx_folders_user_id ON folders (user_id);
CREATE INDEX idx_folders_parent ON folders (parent_folder_id) WHERE parent_folder_id IS NOT NULL;
CREATE INDEX idx_folders_deleted ON folders (user_id, is_deleted) WHERE is_deleted = TRUE;
```

#### `files`

```sql
CREATE TABLE files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    object_key      VARCHAR(512) NOT NULL UNIQUE,          -- opaque MinIO key
    size            BIGINT NOT NULL CHECK (size >= 0),
    mime_type       VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    sha256_hash     VARCHAR(64) NOT NULL,                  -- hex-encoded SHA256
    parent_folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_file_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_file_folder FOREIGN KEY (parent_folder_id) REFERENCES folders(id)
);

CREATE INDEX idx_files_user_id ON files (user_id);
CREATE INDEX idx_files_folder ON files (parent_folder_id);
CREATE INDEX idx_files_folder_listing ON files (parent_folder_id, is_deleted, created_at DESC);
CREATE INDEX idx_files_folder_listing_name ON files (parent_folder_id, is_deleted, name);
CREATE INDEX idx_files_deleted ON files (user_id, is_deleted) WHERE is_deleted = TRUE;
CREATE INDEX idx_files_object_key ON files (object_key);
```

#### `share_links`

```sql
CREATE TABLE share_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type   VARCHAR(20) NOT NULL CHECK (resource_type IN ('file', 'folder')),
    resource_id     UUID NOT NULL,
    token           VARCHAR(64) NOT NULL UNIQUE,            -- opaque share token
    permission      VARCHAR(20) NOT NULL DEFAULT 'preview' CHECK (permission IN ('preview', 'download')),
    expires_at      TIMESTAMPTZ,
    is_revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_share_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_share_links_token ON share_links (token);
CREATE INDEX idx_share_links_user ON share_links (user_id);
CREATE INDEX idx_share_links_resource ON share_links (resource_type, resource_id);
```

#### `refresh_tokens`

```sql
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(255) NOT NULL,                  -- bcrypt hash of token secret only
    expires_at      TIMESTAMPTZ NOT NULL,
    is_revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_refresh_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_user_active ON refresh_tokens (user_id, is_revoked, expires_at);
```

#### Permission model placeholder

MVP authorization is represented by owner fields (`user_id` on `files`/`folders`) plus scoped `share_links.permission`. Do **not** create a separate `permissions`/`acls` table until the folder permission inheritance decision is resolved. The future schema must add ACL entries without changing MinIO object-key rules or bypassing service-layer authorization.

### 9.3 Indexing Strategy

| Table | Index | Purpose |
|---|---|---|
| `users` | `email` (UNIQUE) | Login lookup (exact match) |
| `folders` | `(user_id)` | List user's root folders |
| `folders` | `(parent_folder_id)` | List children of a folder (filtered WHERE NOT NULL) |
| `folders` | `(user_id, is_deleted) WHERE is_deleted` | Recycle bin listing |
| `files` | `(parent_folder_id, is_deleted, created_at DESC)` | File listing sorted by date (primary access pattern) |
| `files` | `(parent_folder_id, is_deleted, name)` | File listing sorted by name (alternative sort) |
| `files` | `(user_id, is_deleted) WHERE is_deleted` | Recycle bin listing |
| `files` | `(object_key)` | MinIO key lookup (rare) |
| `share_links` | `(token)` | Share token lookup (exact match) |
| `share_links` | `(user_id)` | User's share links listing |
| `refresh_tokens` | `PRIMARY KEY (id)` | Refresh token lookup by token ID |
| `refresh_tokens` | `(user_id, is_revoked, expires_at)` | Revoke all by user and find active sessions |

### 9.4 Migration Strategy

Use plain SQL migration files. The migration runner is not fixed here; adding `golang-migrate`, `goose`, or any other tool requires explicit dependency approval first.

**Target file naming convention:**
```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_folders.up.sql
├── 000002_create_folders.down.sql
├── 000003_create_files.up.sql
├── 000003_create_files.down.sql
├── 000004_create_share_links.up.sql
├── 000004_create_share_links.down.sql
├── 000005_create_refresh_tokens.up.sql
└── 000005_create_refresh_tokens.down.sql
```

Create these files incrementally with the feature/migration being implemented; do not pre-create empty migration trees.

---

## 10. API Design

### 10.1 Design Conventions

| Convention | Rule |
|---|---|
| Base URL | `/api/v1/` |
| Request body | JSON (`application/json`) |
| File upload | `multipart/form-data` |
| File download | Streaming response with `Content-Type` from DB metadata |
| Auth header | `Authorization: Bearer <access_token>` |
| Pagination | `?offset=0&limit=20` (default offset=0, limit=20, max limit=100) |
| Error format | `{ "error": { "code": "...", "message": "..." } }` |
| Success format | Direct JSON body (no envelope) for single resources; `{ "data": [...], "total": N }` for lists |
| ID format | UUID (v4) |
| Timestamps | RFC 3339 / ISO 8601 |
| Version header | Not required — version is in URL path |

### 10.2 Error Response Format

**Error response structure:**
```json
{
    "error": {
        "code": "ERROR_CODE",
        "message": "Human-readable description"
    }
}
```

**Standard error codes:**

| HTTP Status | Code | Meaning |
|---|---|---|
| 400 | `INVALID_REQUEST` | Malformed request body or params |
| 400 | `VALIDATION_ERROR` | Field-level validation failed |
| 401 | `MISSING_TOKEN` | No auth token provided |
| 401 | `INVALID_TOKEN` | Token expired or malformed |
| 401 | `TOKEN_REVOKED` | Refresh token has been revoked (replay detected) |
| 403 | `FORBIDDEN` | Authenticated but not authorized |
| 404 | `NOT_FOUND` | Resource not found (generic — used for share link failures too) |
| 409 | `CONFLICT` | Optimistic lock conflict (version mismatch) |
| 409 | `DUPLICATE` | Duplicate name in folder |
| 413 | `FILE_TOO_LARGE` | Upload exceeds size limit |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

**Share link error handling (SHARE-05):**
All share link access failures return the **same generic 404 response**:
```json
{
    "error": {
        "code": "NOT_FOUND",
        "message": "Share link not found or expired"
    }
}
```
This applies to: token not found, expired, revoked, mismatched scope, or resource inaccessible. No timing or error message differentiation.

### 10.3 Pagination

**Request:**
```
GET /api/v1/files?folder_id=<uuid>&offset=0&limit=20&sort=created_at&order=desc
```

**Response:**
```json
{
    "data": [
        { "id": "...", "name": "report.pdf", ... },
        { "id": "...", "name": "photo.jpg", ... }
    ],
    "total": 142
}
```

| Parameter | Default | Max | Description |
|---|---|---|---|
| `offset` | 0 | — | Number of items to skip |
| `limit` | 20 | 100 | Page size |
| `sort` | `created_at` | — | Sort field (`created_at`, `name`) |
| `order` | `desc` | — | Sort direction (`asc`, `desc`) |

### 10.4 Core Endpoint Specifications

#### Authentication

| Method | Path | Handler | Request | Response |
|---|---|---|---|---|
| POST | /api/v1/auth/register | `AuthHandler.Register` | `{ "email": "...", "password": "..." }` | `{ "id": "...", "email": "..." }` |
| POST | /api/v1/auth/login | `AuthHandler.Login` | `{ "email": "...", "password": "..." }` | `{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }` |
| POST | /api/v1/auth/refresh | `AuthHandler.Refresh` | `{ "refresh_token": "..." }` | `{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }` |
| POST | /api/v1/auth/logout | `AuthHandler.Logout` | `{ "refresh_token": "..." }` | 204 No Content |

#### Files

| Method | Path | Handler | Request | Response |
|---|---|---|---|---|
| GET | /api/v1/files | `FileHandler.List` | Query: `folder_id`, `offset`, `limit`, `sort`, `order` | `{ "data": [...], "total": N }` |
| POST | /api/v1/files | `FileHandler.Upload` | Multipart: `file` (binary) + `folder_id` (optional) | 201 + `{ "id": "...", ... }` |
| GET | /api/v1/files/:id | `FileHandler.Get` | — | File metadata JSON |
| GET | /api/v1/files/:id/download | `FileHandler.Download` | Header: `Range` (optional) | Binary stream with Content-Type, Content-Length, Content-Disposition |
| PATCH | /api/v1/files/:id | `FileHandler.Update` | `{ "name": "...", "parent_folder_id": "...", "version": N }` | Updated file metadata |
| DELETE | /api/v1/files/:id | `FileHandler.Delete` | — | 204 No Content |

#### Folders

| Method | Path | Handler | Request | Response |
|---|---|---|---|---|
| POST | /api/v1/folders | `FolderHandler.Create` | `{ "name": "...", "parent_folder_id": "..." }` | 201 + Folder metadata |
| PATCH | /api/v1/folders/:id | `FolderHandler.Update` | `{ "name": "...", "parent_folder_id": "...", "version": N }` | Updated folder metadata |
| DELETE | /api/v1/folders/:id | `FolderHandler.Delete` | — | 204 No Content |

#### Share Links

| Method | Path | Handler | Request | Response |
|---|---|---|---|---|
| POST | /api/v1/shares | `ShareHandler.Create` | `{ "resource_type": "file"/"folder", "resource_id": "...", "permission": "preview"/"download", "expires_in_hours": 24 }` | Share link metadata + URL |
| GET | /api/v1/shares/:token | `ShareHandler.Access` | — | Resource data (file metadata or folder listing) |
| DELETE | /api/v1/shares/:id | `ShareHandler.Revoke` | — | 204 No Content |

#### Recycle Bin

| Method | Path | Handler | Request | Response |
|---|---|---|---|---|
| GET | /api/v1/recycle-bin | `RecycleHandler.List` | Query: `offset`, `limit` | `{ "data": [...], "total": N }` |
| POST | /api/v1/recycle-bin/:id/restore | `RecycleHandler.Restore` | — | Restored resource metadata |
| DELETE | /api/v1/recycle-bin/:id | `RecycleHandler.PermanentDelete` | — | 204 No Content |

#### User

| Method | Path | Handler | Request | Response |
|---|---|---|---|---|
| GET | /api/v1/user/me | `UserHandler.Me` | — | `{ "id": "...", "email": "...", "display_name": "...", "created_at": "..." }` |

---

## 11. Data Flow for Critical Operations

### 11.1 File Upload Flow

```
Browser                     Gin Server                     MinIO              PostgreSQL
  │                            │                             │                   │
  │ POST /api/v1/files         │                             │                   │
  │ (multipart/form-data)      │                             │                   │
  │───────────────────────────→│                             │                   │
  │                            │                             │                   │
  │                            │ 1. Auth middleware validates JWT                  │
  │                            │                             │                   │
  │                            │ 2. Rate limiter checks quota                      │
  │                            │                             │                   │
  │                            │ 3. Handler reads Content-Length                  │
  │                            │    If > max file size → 413                      │
  │                            │                             │                   │
  │                            │ 4. Handler parses multipart form                 │
  │                            │    Extracts file + folder_id                     │
  │                            │                             │                   │
  │                            │ 5. Handler→FileService.Upload()                  │
  │                            │                             │                   │
  │                            │ 6. Generate UUID for object_key                  │
  │                            │                             │                   │
  │                            │ 7. Detect MIME type from first 512 bytes         │
  │                            │                             │                   │
  │                            │ 8. Compute SHA256 while streaming                │
  │                            │                             │                   │
  │                            │ 9. Stream to MinIO         │                   │
  │                            │───────────────────────────→│                   │
  │                            │    io.Copy with TeeReader   │                   │
  │                            │    (hash + upload in one    │                   │
  │                            │     pass, no buffering)     │                   │
  │                            │                             │                   │
  │                            │    ←── Success ───────────│                   │
  │                            │                             │                   │
  │                            │ 10. Insert metadata into DB │                   │
  │                            │───────────────────────────────────────────────→│
  │                            │                             │                   │
  │                            │    ←── Success ───────────────────────────────│
  │                            │                             │                   │
  │                            │ [If DB fails:]              │                   │
  │                            │ 11. Delete MinIO object     │                   │
  │                            │───────────────────────────→│                   │
  │                            │    Cleanup to avoid orphans │                   │
  │                            │                             │                   │
  │  ←── 201 + file metadata ──│                             │                   │
  │                            │                             │                   │
```

**Key design decisions:**
- Hash computation and upload happen in a single pass using `io.TeeReader` — no separate hash pass
- MIME type detected server-side from first bytes (using `net/http`'s `DetectContentType`)
- File size limit enforced **before** reading the multipart body, using `MaxBytesReader` in Gin or `Content-Length` check
- MinIO upload happens BEFORE DB insert (MinIO is the primary — if it fails, no DB work needed; if DB fails, cleanup MinIO)

### 11.2 File Download Flow

```
Browser                     Gin Server                     MinIO
  │                            │                             │
  │ GET /api/v1/files/:id      │                             │
  │ /download                  │                             │
  │───────────────────────────→│                             │
  │                            │                             │
  │                            │ 1. Auth middleware validates JWT
  │                            │                             │
  │                            │ 2. Handler→FileService.Download()
  │                            │                             │
  │                            │ 3. Service: FileRepository  │
  │                            │    .GetByID()               │
  │                            │        │                    │
  │                            │        ▼                    │
  │                            │    - File exists?           │
  │                            │    - User owns it?          │
  │                            │    - Not deleted?           │
  │                            │                             │
  │                            │ 4. Service→FileStorage      │
  │                            │    .Download(object_key)    │
  │                            │───────────────────────────→│
  │                            │                             │
  │                            │    ←── io.ReadCloser ──────│
  │                            │                             │
  │                            │ 5. Handler sets response headers:
  │                            │    Content-Type: mime_type
  │                            │    Content-Length: size
  │                            │    Content-Disposition: attachment; filename="..."
  │                            │    Accept-Ranges: bytes
  │                            │                             │
  │                            │ 6. Handler streams via      │
  │                            │    io.Copy(c.Writer, reader)│
  │                            │                             │
  │  ←── Binary stream ────────│                             │
  │                            │                             │
```

**Range request support (for file preview in future):**
When the client sends a `Range` header, the handler should:
1. Parse the range header
2. Use MinIO's `GetObject` with `GetObjectOptions{Offset: start, Length: end-start+1}`
3. Respond with `206 Partial Content` + `Content-Range` header

### 11.3 Soft-Delete + Recycle Bin Flow

```
User action: "Delete file"
  │
  ▼
FileService.Delete(ctx, userID, fileID)
  │
  ├── 1. FileRepository.GetByID(fileID)
  │       └── Check: exists + user owns it + not already deleted
  │
  ├── 2. FileRepository.SoftDelete(fileID)
  │       └── UPDATE files SET is_deleted=true, deleted_at=NOW() WHERE id=$1
  │
  └── Return success (204)
```

**Restore flow:**
```
User action: "Restore file from recycle bin"
  │
  ▼
RecycleService.Restore(ctx, userID, itemID)
  │
  ├── 1. Determine if item is file or folder
  │
  ├── 2. Repository: Get the deleted item
  │       └── Check: exists + user owns it + is_deleted
  │
  ├── 3. If parent_folder_id is also deleted:
  │       └── Restore parent folder first (recursive)
  │       └── OR restore to root (if parent permanently deleted)
  │
  ├── 4. Repository: UPDATE item SET is_deleted=false, deleted_at=NULL
  │
  └── Return restored item metadata
```

### 11.4 Folder Delete Cascade Flow

```
FolderService.Delete(ctx, userID, folderID)
  │
  ├── 1. FolderRepository.GetByID(folderID)
  │       └── Check: exists + user owns it + not deleted
  │
  ├── 2. Get all descendant folder IDs (recursive CTE)
  │
  ├── 3. Begin DB transaction
  │
  ├── 4. Soft-delete all sub-folders (set is_deleted, deleted_at)
  │
  ├── 5. Soft-delete all files in those folders
  │
  ├── 6. Soft-delete the target folder itself
  │
  ├── 7. Commit transaction
  │
  └── Return success (204)

Note: No MinIO operations during soft-delete.
MinIO objects are only removed during PERMANENT delete.
```

**Permanent delete cascade (BIN-04, BIN-05):**
```
RecycleService.PermanentDelete(ctx, userID, folderID)
  │
  ├── 1. Get all file object_keys in the subtree
  │
  ├── 2. For each object_key: FileStorage.Delete(ctx, object_key)
  │       └── Log errors but continue (BIN-05: handle partial failures)
  │
  ├── 3. DB transaction:
  │       ├── Delete all file records in subtree
  │       ├── Delete all folder records in subtree
  │       └── Commit
  │
  └── Return success
```

### 11.5 Share Link Access Flow

```
Browser                     Gin Server                     PostgreSQL
  │                            │                             │
  │ GET /api/v1/shares/:token  │                             │
  │───────────────────────────→│                             │
  │                            │                             │
  │                            │ 1. No JWT check — public endpoint
  │                            │                             │
  │                            │ 2. Query share_links by token
  │                            │────────────────────────────→│
  │                            │                             │
  │                            │    ←── result or not found ─│
  │                            │                             │
  │                            │ 3. Validate:
  │                            │    - Token exists?
  │                            │    - Not expired?
  │                            │    - Not revoked?
  │                            │    - Permission allows access?
  │                            │                             │
  │                            │ [ANY failure → 404 with generic message]
  │                            │                             │
  │                            │ 4. Look up the resource:
  │                            │    - If file: return file metadata; only stream content through an approved preview/download path when scope allows
  │                            │    - If folder: return folder contents listing
  │                            │                             │
  │                            │    ←── resource data ──────│
  │                            │                             │
  │  ←── 200 + resource data ──│                             │
  │                            │                             │
```

### 11.6 File Rename with Optimistic Locking

```
Client                              Server
  │                                   │
  │ GET /api/v1/files/:id             │
  │←── { ..., version: 5 } ──────────│  (read current version)
  │                                   │
  │ PATCH /api/v1/files/:id           │
  │ { "name": "new-name.pdf",         │
  │   "version": 5 }                  │
  │─────────────────────────────────→│
  │                                   │
  │                                   │ UPDATE files
  │                                   │ SET name = 'new-name.pdf',
  │                                   │     version = version + 1,
  │                                   │     updated_at = NOW()
  │                                   │ WHERE id = $1
  │                                   │   AND version = 5
  │                                   │
  │                                   │ [If rows_affected == 0]
  │                                   │ → 409 CONFLICT
  │                                   │   { "code": "CONFLICT",
  │                                   │     "message": "File was modified by another request" }
  │                                   │
  │  ←── 409 or updated metadata ────│
```

---

## 12. Error Handling Strategy

### 12.1 Error Types

```go
// model/errors.go
package model

import "errors"

// Sentinel errors for domain-level failures
var (
    ErrNotFound         = errors.New("resource not found")
    ErrForbidden        = errors.New("access denied")
    ErrConflict         = errors.New("version conflict")
    ErrDuplicate        = errors.New("duplicate resource")
    ErrInvalidInput     = errors.New("invalid input")
    ErrTokenRevoked     = errors.New("token has been revoked")
    ErrFileTooLarge     = errors.New("file exceeds size limit")
    ErrRateLimited      = errors.New("rate limit exceeded")
)

// DomainError wraps a sentinel error with additional context
type DomainError struct {
    Err     error
    Code    string   // machine-readable error code
    Message string   // user-safe message
}

func (e *DomainError) Error() string { return e.Err.Error() }
func (e *DomainError) Unwrap() error  { return e.Err }

func NewNotFoundError(msg string) *DomainError {
    return &DomainError{Err: ErrNotFound, Code: "NOT_FOUND", Message: msg}
}

func NewForbiddenError(msg string) *DomainError {
    return &DomainError{Err: ErrForbidden, Code: "FORBIDDEN", Message: msg}
}

func NewConflictError(msg string) *DomainError {
    return &DomainError{Err: ErrConflict, Code: "CONFLICT", Message: msg}
}

func NewInvalidInputError(msg string) *DomainError {
    return &DomainError{Err: ErrInvalidInput, Code: "INVALID_REQUEST", Message: msg}
}
```

### 12.2 Error Propagation

```
Repository ──→ Service ──→ Handler ──→ HTTP Response

Repository:
  - Returns wrapped errors: fmt.Errorf("query get file: %w", model.ErrNotFound)
  - Never returns SQL-specific errors directly

Service:
  - Converts repository/storage errors to DomainError
  - Wraps with context: model.NewNotFoundError("File not found")
  - Never passes raw SQL or MinIO errors upward

Handler:
  - Uses httputil.RespondServiceError() to map DomainError → HTTP status
  - Unexpected errors → 500 Internal Server Error (logged, not exposed)
```

### 12.3 Global Error Handler

```go
// pkg/httputil/response.go

func RespondServiceError(c *gin.Context, err error) {
    var domainErr *model.DomainError
    if errors.As(err, &domainErr) {
        status := domainErrToHTTPStatus(domainErr)
        RespondError(c, status, domainErr.Code, domainErr.Message)
        return
    }

    // Unknown error — log and return generic 500
    log.Printf("unexpected error: %v", err)
    RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}

func domainErrToHTTPStatus(err *model.DomainError) int {
    switch {
    case errors.Is(err.Err, model.ErrNotFound):
        return http.StatusNotFound
    case errors.Is(err.Err, model.ErrForbidden):
        return http.StatusForbidden
    case errors.Is(err.Err, model.ErrConflict):
        return http.StatusConflict
    case errors.Is(err.Err, model.ErrDuplicate):
        return http.StatusConflict
    case errors.Is(err.Err, model.ErrInvalidInput):
        return http.StatusBadRequest
    case errors.Is(err.Err, model.ErrTokenRevoked):
        return http.StatusUnauthorized
    case errors.Is(err.Err, model.ErrFileTooLarge):
        return http.StatusRequestEntityTooLarge
    case errors.Is(err.Err, model.ErrRateLimited):
        return http.StatusTooManyRequests
    default:
        return http.StatusInternalServerError
    }
}
```

---

## 13. Scheduled Tasks

### 13.1 Recycle Bin Cleanup

**Requirement (BIN-07, NFR-REL-03):** A configurable periodic process permanently removes expired recycle bin items (items where `deleted_at` exceeds a configurable TTL, e.g., 30 days).

**Behavior:**
1. Find all items where `is_deleted = true` AND `deleted_at < NOW() - INTERVAL '$1 days'`
2. For files: delete MinIO objects, then delete DB records
3. For folders: cascade to all sub-items
4. Handle partial MinIO failures gracefully — log error, do not block subsequent deletions

### 13.2 Architecture

```go
// internal/cleanup/recycle_bin.go
type RecycleBinCleaner struct {
    fileRepo   service.FileRepository
    folderRepo service.FolderRepository
    storage    service.FileStorage
    ttl        time.Duration     // configurable, default 30 days
    interval   time.Duration     // configurable, default 24 hours
}

func (c *RecycleBinCleaner) Start(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.cleanup(ctx)
        }
    }
}

func (c *RecycleBinCleaner) cleanup(ctx context.Context) {
    // 1. Find expired items
    files, folders, err := c.findExpiredItems(ctx)
    if err != nil {
        log.Printf("cleanup: failed to find expired items: %v", err)
        return
    }

    // 2. Delete file objects from MinIO
    for _, f := range files {
        if err := c.storage.Delete(ctx, f.ObjectKey); err != nil {
            log.Printf("cleanup: failed to delete minio object %s: %v", f.ObjectKey, err)
            // Continue — don't block cleanup on single failure (BIN-07)
        }
    }

    // 3. Delete DB records (in transaction)
    if err := c.deleteExpiredRecords(ctx, files, folders); err != nil {
        log.Printf("cleanup: failed to delete expired records: %v", err)
        // Items remain in DB with is_deleted=true; will be retried next cycle.
    }
}
```

**Cleanup is started in `cmd/server/main.go`:**
```go
cleaner := cleanup.NewRecycleBinCleaner(fileRepo, folderRepo, storage, cfg.CleanupTTL, cfg.CleanupInterval)
go cleaner.Start(ctx)
```

---

## 14. Frontend Architecture

### 14.1 Directory Structure

```
frontend/
├── src/
│   ├── components/           # Reusable UI components
│   │   ├── common/           # Button, Input, Modal, Spinner, Pagination
│   │   ├── file/             # FileIcon, FileList, FileCard, UploadProgress
│   │   ├── folder/           # FolderTree, FolderBreadcrumb
│   │   ├── share/            # ShareLinkDialog, ShareLinkList
│   │   └── layout/           # Sidebar, Header, MainLayout
│   ├── pages/                # Route-level page components
│   │   ├── LoginPage.tsx
│   │   ├── RegisterPage.tsx
│   │   ├── DashboardPage.tsx
│   │   ├── RecycleBinPage.tsx
│   │   ├── SharedPage.tsx    # Share link access view
│   │   └── NotFoundPage.tsx
│   ├── hooks/                # Custom React hooks
│   │   ├── useAuth.ts
│   │   ├── useFiles.ts
│   │   ├── useFolders.ts
│   │   ├── useUpload.ts      # Upload with progress tracking
│   │   └── useShareLinks.ts
│   ├── services/             # API client layer
│   │   ├── api.ts            # Base HTTP client with auth interceptor
│   │   ├── auth.ts           # Auth API calls
│   │   ├── files.ts          # File API calls
│   │   ├── folders.ts        # Folder API calls
│   │   ├── shares.ts         # Share link API calls
│   │   └── recycleBin.ts     # Recycle bin API calls
│   ├── store/                # State management
│   │   ├── AuthContext.tsx
│   │   ├── FileContext.tsx
│   │   └── UIContext.tsx
│   ├── types/                # TypeScript type definitions
│   │   ├── api.ts            # API response types
│   │   ├── models.ts         # Domain models (File, Folder, User, ShareLink)
│   │   └── common.ts         # Pagination, error types
│   └── utils/
│       ├── format.ts         # File size formatting, date formatting
│       └── validation.ts     # Form validation helpers
├── public/
├── package.json
├── tsconfig.json
└── vite.config.ts
```

### 14.2 Component Hierarchy

```
<App>
  <AuthProvider>
    <UIProvider>
      <FileProvider>                // current folder context
        <Router>
          <MainLayout>
            <Sidebar>               // FolderTree, RecycleBin link, Logout
            <Header>                // Breadcrumb, Search, User menu
            <Content>
              <Routes>
                <LoginPage />
                <RegisterPage />
                <DashboardPage>
                  <Toolbar />       // New folder, Upload button
                  <Breadcrumb />
                  <FileList>        // or grid view
                    <FileRow />     // icon, name, size, date, actions
                    <FileRow />
                  </FileList>
                  <Pagination />
                  <UploadProgress />
                </DashboardPage>
                <RecycleBinPage>
                  <RecycleList />
                </RecycleBinPage>
                <SharedPage />      // Public share link view
                <NotFoundPage />
              </Routes>
            </Content>
          </MainLayout>
        </Router>
      </FileProvider>
    </UIProvider>
  </AuthProvider>
</App>
```

### 14.3 State Management

Use **React Context + useReducer** for MVP. No external state management library (Redux, Zustand) without explicit approval.

**Three contexts:**

1. **AuthContext** — user session state
   ```typescript
   interface AuthState {
       user: User | null;
       accessToken: string | null;
       refreshToken: string | null;
       isAuthenticated: boolean;
       isLoading: boolean;
   }
   ```

2. **FileContext** — current folder navigation state
   ```typescript
   interface FileState {
       currentFolder: Folder | null;
       breadcrumbs: Breadcrumb[];
       files: File[];
       folders: Folder[];
       total: number;
       offset: number;
       limit: number;
       isLoading: boolean;
       selectedItems: Set<string>;
   }
   ```

3. **UIContext** — modal/dialog state, notifications
   ```typescript
   interface UIState {
       uploadProgress: Map<string, number>;
       notifications: Notification[];
       modals: {
           shareLink: { open: boolean; item?: File | Folder };
           confirmDelete: { open: boolean; item?: File | Folder };
           renameItem: { open: boolean; item?: File | Folder };
       };
   }
   ```

### 14.4 API Client Layer

```typescript
// services/api.ts
class ApiClient {
    private baseUrl: string;
    private getAccessToken: () => string | null;
    private onTokenRefresh: () => Promise<string>;

    async request<T>(method: string, path: string, options?: RequestOptions): Promise<T> {
        const token = this.getAccessToken();
        const headers: HeadersInit = {
            'Authorization': token ? `Bearer ${token}` : '',
        };

        let response = await fetch(`${this.baseUrl}${path}`, {
            method,
            headers,
            body: options?.body,
        });

        // Token expired — attempt refresh
        if (response.status === 401 && token) {
            const newToken = await this.onTokenRefresh();
            headers['Authorization'] = `Bearer ${newToken}`;
            response = await fetch(`${this.baseUrl}${path}`, {
                method, headers, body: options?.body,
            });
        }

        if (!response.ok) {
            const err = await response.json();
            throw new ApiError(response.status, err.error?.code, err.error?.message);
        }

        return response.json();
    }

    // Streaming download
    downloadFile(path: string): Promise<Response> {
        return fetch(`${this.baseUrl}${path}`, {
            headers: { 'Authorization': `Bearer ${this.getAccessToken()}` },
        });
    }
}
```

**Upload with progress:**
```typescript
// hooks/useUpload.ts
async function uploadFile(file: File, folderId: string, onProgress: (pct: number) => void) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('folder_id', folderId);

    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.upload.onprogress = (e) => {
            if (e.lengthComputable) onProgress(Math.round((e.loaded / e.total) * 100));
        };
        xhr.onload = () => { ... };
        xhr.onerror = () => reject(new Error('Upload failed'));
        xhr.open('POST', '/api/v1/files');
        xhr.setRequestHeader('Authorization', `Bearer ${getAccessToken()}`);
        xhr.send(formData);
    });
}
```

### 14.5 Route Design

```typescript
// App.tsx
<Routes>
    <Route path="/login" element={<LoginPage />} />
    <Route path="/register" element={<RegisterPage />} />
    <Route path="/" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
    <Route path="/folder/:folderId" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
    <Route path="/recycle-bin" element={<ProtectedRoute><RecycleBinPage /></ProtectedRoute>} />
    <Route path="/s/:token" element={<SharedPage />} />    {/* share link — no auth */}
    <Route path="*" element={<NotFoundPage />} />
</Routes>
```

**ProtectedRoute** — redirects to `/login` if not authenticated.

---

## 15. Local Development Architecture

### 15.1 Development Environment (Docker Compose)

```yaml
# docker-compose.yml (development)
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: nuonetdisk
      POSTGRES_USER: nuonetdisk
      POSTGRES_PASSWORD: devpassword
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nuonetdisk"]
      interval: 5s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: nuonetdisk
      MINIO_ROOT_PASSWORD: devpassword123
    ports:
      - "9000:9000"    # S3 API
      - "9001:9001"    # Console UI
    volumes:
      - miniodata:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: .
      dockerfile: backend/Dockerfile
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: nuonetdisk
      DB_PASSWORD: devpassword
      DB_NAME: nuonetdisk
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: nuonetdisk
      MINIO_SECRET_KEY: devpassword123
      MINIO_BUCKET: nuonetdisk
      JWT_SECRET: dev-jwt-secret-change-in-production
      JWT_ACCESS_TTL: 15m
      JWT_REFRESH_TTL: 168h  # 7 days
      SERVER_PORT: 8080
      UPLOAD_MAX_SIZE: 524288000  # 500MB
      CLEANUP_TTL: 720h           # 30 days
      CLEANUP_INTERVAL: 24h
      CORS_ORIGINS: http://localhost:5173
      GIN_MODE: debug
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy

  frontend:
    build:
      context: .
      dockerfile: frontend/Dockerfile
    ports:
      - "80:80"       # Served via Nginx
    depends_on:
      - backend

volumes:
  pgdata:
  miniodata:
```

### 15.2 Service Configuration

This section is local development only. It does not approve production infrastructure or require Nginx for MVP.

**Optional local-dev frontend static server / API proxy example:**
```nginx
server {
    listen 80;
    server_name localhost;

    # Frontend static files
    root /usr/share/nginx/html;
    index index.html;

    # SPA — serve index.html for all frontend routes
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_request_buffering off;   # Essential for streaming uploads
        proxy_buffering off;           # Essential for streaming downloads
        client_max_body_size 0;        # Backend handles size limits
    }
}
```

### 15.3 Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | Backend HTTP listen port |
| `GIN_MODE` | `release` | Gin mode (`debug`, `release`, `test`) |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `nuonetdisk` | PostgreSQL user |
| `DB_PASSWORD` | — | PostgreSQL password |
| `DB_NAME` | `nuonetdisk` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `DB_MAX_OPEN_CONNS` | `25` | Database connection pool max open |
| `DB_MAX_IDLE_CONNS` | `10` | Database connection pool max idle |
| `MINIO_ENDPOINT` | `localhost:9000` | MinIO server endpoint |
| `MINIO_ACCESS_KEY` | — | MinIO access key |
| `MINIO_SECRET_KEY` | — | MinIO secret key |
| `MINIO_BUCKET` | `nuonetdisk` | MinIO bucket name |
| `MINIO_USE_SSL` | `false` | Use HTTPS for MinIO |
| `JWT_SECRET` | — | HMAC signing key (min 32 chars) |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (7 days) |
| `UPLOAD_MAX_SIZE` | `524288000` | Max upload size in bytes (500MB) |
| `CLEANUP_TTL` | `720h` | Recycle bin retention period (30 days) |
| `CLEANUP_INTERVAL` | `24h` | Cleanup job interval |
| `CORS_ORIGINS` | — | Allowed CORS origins (comma-separated); no wildcard default outside local development |
| `RATE_LIMIT_IP_REQS` | `100` | Per-IP requests/second |
| `RATE_LIMIT_IP_BURST` | `200` | Per-IP burst |
| `RATE_LIMIT_USER_REQS` | `1000` | Per-user requests/minute |
| `LOG_LEVEL` | `info` | Logging level |

### 15.4 Network Architecture (Development)

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Network                         │
│                                                          │
│  ┌──────────┐    :5432    ┌───────────┐                  │
│  │PostgreSQL│◄────────────│           │                  │
│  └──────────┘             │  Backend  │                  │
│                           │  (Go/Gin) │                  │
│  ┌──────────┐    :9000    │  :8080    │                  │
│  │  MinIO   │◄────────────│           │                  │
│  └──────────┘             └─────┬─────┘                  │
│                                 │ :8080                   │
│                                 ▼                         │
│                          ┌──────────┐                    │
│                          │  Nginx   │ :80                 │
│                          │ (Reverse │                    │
│                          │  Proxy)  │                    │
│                          └────┬─────┘                    │
│                               │                          │
│                               ▼                          │
│                          Host Machine                    │
│                     Browser → http://localhost:80         │
└─────────────────────────────────────────────────────────┘
```

**Port mapping summary:**

| Service | Internal Port | External Port |
|---|---|---|
| Nginx (frontend) | 80 | 80 |
| Backend (Gin) | 8080 | 8080 (direct, for debugging) |
| PostgreSQL | 5432 | 5432 |
| MinIO S3 API | 9000 | 9000 |
| MinIO Console | 9001 | 9001 |

---

## 16. Security Architecture

### 16.1 Defense in Depth

| Layer | Control | Requirement |
|---|---|---|
| **Network** | CORS restrictions | NFR-SEC-10 |
| **HTTP** | Rate limiting (per-IP + per-user) | NFR-SEC-09, NFR-SEC-12 |
| **Auth** | JWT access tokens (short-lived) | AUTH-03 |
| **Auth** | Refresh token rotation + replay detection | AUTH-09 |
| **Auth** | bcrypt password hashing (cost >= 10) | AUTH-07, NFR-SEC-03 |
| **Handler** | Input validation (size, format, path) | FILE-03, FILE-13 |
| **Handler** | Server-side MIME detection | FILE-06 |
| **Service** | Resource ownership check (user_id match) | FILE-12 |
| **Service** | Share link token validation + expiry + revocation | SHARE-05 |
| **Service** | Optimistic locking for rename/move | NFR-SEC-13 |
| **Repository** | Parameterized SQL queries | NFR-SEC-04 |
| **Storage** | Opaque object keys (UUID-based) | FILE-04, FILE-05 |
| **Logging** | Sensitive data redaction | NFR-SEC-02, NFR-SEC-14 |
| **Response** | Uniform error responses (no information leakage) | SHARE-05, NFR-SEC-11 |
| **Infrastructure** | File size limit before reading body | NFR-SEC-07 |

### 16.2 Threat Model Summary

| Threat | Mitigation |
|---|---|
| Unauthorized file access | JWT auth on all endpoints except whitelist; resource ownership check in service layer |
| Share link enumeration | Opaque tokens (high-entropy random); same 404 for all failure cases |
| Token theft (access) | Short TTL (15 min); no server-side state needed |
| Token theft (refresh) | Rotation + revocation; replay detection revokes all sessions |
| Path traversal | Strip/reject `../` sequences; store only sanitized filename; opaque MinIO keys |
| Upload bomb (huge file) | Backend Content-Length / MaxBytesReader checks before body read; any local proxy must avoid buffering uploads |
| SQL injection | Parameterized queries only |
| Timing side-channel | bcrypt password comparison (constant-time); same error response for share link failures |
| MinIO key exposure | Never sent to frontend; backend mediates all storage access |
| Dual-write inconsistency | Upload→MinIO first; on DB failure→cleanup MinIO object |
| Brute force login | Rate limiting on auth endpoints; bcrypt cost >= 10 |
| Log leakage | Redaction middleware; no tokens/passwords in any log output |

---

## 17. Pending Decisions & Future Considerations

The following decisions are **deferred** per AGENTS.md §8.3. The architecture above accommodates them without pre-implementation:

| Decision | Impact | Current State | Future Action |
|---|---|---|---|
| **File preview approach** | Server-side conversion (LibreOffice, ffmpeg) or browser-native preview | No approved approach yet; browser-native preview is only a candidate | Resolve design first; only then implement approved preview endpoints with streaming/range support |
| **Folder permission inheritance model** | Closure table vs. path enumeration vs. adjacency-list with recursive CTEs | MVP uses resource-owner only + share links; ACL table is deferred | Design the ACL system, then add `permissions`/`acls` schema and implement inheritance |
| **Infrastructure (CI/CD, deployment)** | GitHub Actions, automated tests, staging/prod environments | Docker Compose for local dev only | Do not add production/CI infrastructure until approved |
| **Storage quota enforcement** | Per-user storage limits | Not implemented (out of scope for MVP) | Add `storage_quota` and `storage_used` to users table |
| **File versioning** | Version history for files | Optimistic locking column (`version`) is reserved for concurrency, not version history | Add versioning table or extend file schema |

---

## Appendix A: Go Dependency Justification

Per AGENTS.md Dependency Policy, no Go module or npm package is pre-approved merely because the technology stack is fixed. Each dependency requires explicit approval before it is added.

| Dependency | Purpose | Alternatives | Status | Notes |
|---|---|---|---|---|
| `github.com/gin-gonic/gin` | HTTP framework | net/http, echo, chi | Approval required | Gin is the fixed framework choice, but the module addition still needs explicit approval |
| PostgreSQL driver (`github.com/jackc/pgx/v5` or equivalent) | PostgreSQL connectivity | database/sql + driver, pgx | Approval required | Pick one during backend setup and justify it |
| MinIO SDK (`github.com/minio/minio-go/v7` or equivalent) | MinIO object operations | AWS SDK S3 client | Approval required | Prefer the official MinIO client if approved |
| JWT library (`github.com/golang-jwt/jwt/v5` or equivalent) | JWT signing/validation | standard crypto + manual claims validation, jwx | Approval required | Choose during auth implementation |
| `golang.org/x/crypto` | bcrypt password hashing | approved equivalent password-hashing package | Approval required | Required if bcrypt is implemented with Go x/crypto |
| Migration tool (`golang-migrate`, `goose`, or equivalent) | DB migration runner | manual SQL runner | Approval required | Plain SQL files are allowed; tool choice is deferred |
| UUID library (`github.com/google/uuid` or equivalent) | UUID generation | standard library `crypto/rand` | Approval required | Generated object keys must remain opaque |
| Structured logging library | Structured logs | standard `log/slog` | Approval required | Prefer `log/slog` unless a dependency is justified |
| Env var loader | Configuration loading | standard `os.Getenv` | Approval required | Prefer standard library unless a dependency is justified |

## Appendix B: Key Go Interface Summary

```go
// ========== Handler → Service ==========

// service/auth.go
type AuthService interface {
    Register(ctx context.Context, email, password string) (*model.User, error)
    Login(ctx context.Context, email, password string) (*AuthTokens, error)
    Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
    Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error
}

// service/file.go
type FileService interface {
    List(ctx context.Context, userID, folderID uuid.UUID, offset, limit int, sort, order string) ([]model.File, int, error)
    Upload(ctx context.Context, userID, folderID uuid.UUID, filename string, size int64, mimeType string, reader io.Reader) (*model.File, error)
    Get(ctx context.Context, userID, fileID uuid.UUID) (*model.File, error)
    Download(ctx context.Context, userID, fileID uuid.UUID) (*model.File, io.ReadCloser, error)
    Update(ctx context.Context, userID, fileID uuid.UUID, req UpdateFileRequest) (*model.File, error)
    Delete(ctx context.Context, userID, fileID uuid.UUID) error
}

// service/folder.go
type FolderService interface {
    Create(ctx context.Context, userID, parentFolderID uuid.UUID, name string) (*model.Folder, error)
    Update(ctx context.Context, userID, folderID uuid.UUID, req UpdateFolderRequest) (*model.Folder, error)
    Delete(ctx context.Context, userID, folderID uuid.UUID) error
}

// service/share.go
type ShareService interface {
    Create(ctx context.Context, userID uuid.UUID, req CreateShareRequest) (*model.ShareLink, error)
    Access(ctx context.Context, token string) (*ShareResource, error)
    Revoke(ctx context.Context, userID, shareID uuid.UUID) error
}

// service/recycle_bin.go
type RecycleBinService interface {
    List(ctx context.Context, userID uuid.UUID, offset, limit int) ([]RecycleBinItem, int, error)
    Restore(ctx context.Context, userID, itemID uuid.UUID, itemType string) (any, error)
    PermanentDelete(ctx context.Context, userID, itemID uuid.UUID, itemType string) error
}

// ========== Service → Repository ==========

// Defined in each service file:
// e.g., service/file.go
type FileRepository interface {
    Insert(ctx context.Context, f *model.File) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.File, error)
    ListByFolder(ctx context.Context, userID, folderID uuid.UUID, offset, limit int, sort, order string) ([]model.File, int, error)
    Update(ctx context.Context, f *model.File) error
    SoftDelete(ctx context.Context, id uuid.UUID) error
    GetDeleted(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.File, int, error)
    Restore(ctx context.Context, id uuid.UUID) error
    PermanentDelete(ctx context.Context, id uuid.UUID) error
    GetByObjectKey(ctx context.Context, objectKey string) (*model.File, error)
}

// ========== Service → Storage ==========

// Defined in service/file.go or service/storage.go
type FileStorage interface {
    Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
    Download(ctx context.Context, objectKey string) (io.ReadCloser, error)
    Delete(ctx context.Context, objectKey string) error
    Copy(ctx context.Context, srcKey, dstKey string) error
}
```
