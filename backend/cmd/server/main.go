package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuonuo/nuonetdisk/internal/auth"
	"github.com/nuonuo/nuonetdisk/internal/config"
	"github.com/nuonuo/nuonetdisk/internal/handler"
	"github.com/nuonuo/nuonetdisk/internal/middleware"
	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/internal/storage"
)

var Version = "dev" // overridden by -ldflags at build time

func main() {
	cfg := config.MustLoad()

	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	minioStorage, err := storage.NewMinioStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		log.Fatalf("failed to create MinIO storage: %v", err)
	}

	runMigrations(dbPool)

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTAccessExpiry)
	refreshSvc := auth.NewRefreshTokenService()

	userRepo := repository.NewUserRepository(dbPool)
	fileRepo := repository.NewFileRepository(dbPool)
	folderRepo := repository.NewFolderRepository(dbPool)
	shareRepo := repository.NewShareRepository(dbPool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(dbPool)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtService, refreshSvc, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)
	userService := service.NewUserService(userRepo)
	fileService := service.NewFileService(fileRepo, folderRepo, minioStorage, cfg.MaxUploadSize)
	folderService := service.NewFolderService(folderRepo, fileRepo, minioStorage)
	shareService := service.NewShareService(shareRepo, fileRepo, folderRepo, cfg.MaxShareTTL)
	recycleBinService := service.NewRecycleBinService(fileRepo, folderRepo, minioStorage)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	fileHandler := handler.NewFileHandler(fileService)
	folderHandler := handler.NewFolderHandler(folderService)
	shareHandler := handler.NewShareHandler(shareService, fileService, folderService)
	recycleBinHandler := handler.NewRecycleBinHandler(recycleBinService)

	r := gin.New()
	middleware.Apply(r, jwtService, cfg.CORSAllowedOrigins, middleware.RateLimitConfig{
		IPRate:    float64(cfg.RateLimitGeneral),
		IPBurst:   cfg.RateLimitGeneral,
		UserRate:  float64(cfg.RateLimitGeneral) * 10,
		UserBurst: cfg.RateLimitGeneral * 10,
		AuthRate:  float64(cfg.RateLimitAuth),
		AuthBurst: cfg.RateLimitAuth,
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		user := api.Group("/user")
		{
			user.GET("/me", userHandler.GetMe)
			user.PATCH("/me", userHandler.UpdateMe)
		}

		files := api.Group("/files")
		{
			files.GET("", fileHandler.List)
			files.POST("", fileHandler.Upload)
			files.GET("/:fileId", fileHandler.GetByID)
			files.GET("/:fileId/download", fileHandler.Download)
			files.PATCH("/:fileId", fileHandler.Update)
			files.DELETE("/:fileId", fileHandler.Delete)
		}

		folders := api.Group("/folders")
		{
			folders.POST("", folderHandler.Create)
			folders.GET("", folderHandler.List)
			folders.GET("/resolve", folderHandler.ResolveByName)
			folders.GET("/resolve-by-path", folderHandler.ResolveByPath)
			folders.GET("/:folderId", folderHandler.GetByID)
			folders.GET("/:folderId/ancestors", folderHandler.GetAncestors)
			folders.PATCH("/:folderId", folderHandler.Update)
			folders.DELETE("/:folderId", folderHandler.Delete)
		}

		shares := api.Group("/shares")
		{
			shares.POST("", shareHandler.Create)
			shares.DELETE("/:shareId", shareHandler.Revoke)
			shares.GET("/token/:token", shareHandler.AccessByToken)
		}

		recycleBin := api.Group("/recycle-bin")
		{
			recycleBin.GET("/files", recycleBinHandler.ListDeletedFiles)
			recycleBin.GET("/folders", recycleBinHandler.ListDeletedFolders)
			recycleBin.POST("/restore/file/:fileId", recycleBinHandler.RestoreFile)
			recycleBin.POST("/restore/folder/:folderId", recycleBinHandler.RestoreFolder)
			recycleBin.DELETE("/file/:fileId", recycleBinHandler.PermanentDeleteFile)
			recycleBin.DELETE("/folder/:folderId", recycleBinHandler.PermanentDeleteFolder)
		}
	}

	cleanupSvc := service.NewCleanupService(dbPool, cfg.RecycleBinCleanupInterval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cleanupSvc.Start(ctx)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func runMigrations(pool *pgxpool.Pool) {
	log.Println("running database migrations...")

	migrationsDir := "migrations"
	if _, err := os.Stat("/migrations"); err == nil {
		migrationsDir = "/migrations"
	}

	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		log.Fatalf("failed to create schema_migrations table: %v", err)
	}

	rows, err := pool.Query(context.Background(), "SELECT filename FROM schema_migrations ORDER BY filename")
	if err != nil {
		log.Fatalf("failed to query applied migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("failed to scan migration filename: %v", err)
		}
		applied[name] = true
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("failed to read migrations directory: %v", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		if applied[entry.Name()] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			log.Fatalf("failed to read migration %s: %v", entry.Name(), err)
		}

		tx, err := pool.Begin(context.Background())
		if err != nil {
			log.Fatalf("failed to begin transaction: %v", err)
		}

		if _, err := tx.Exec(context.Background(), string(content)); err != nil {
			_ = tx.Rollback(context.Background())
			log.Fatalf("failed to execute migration %s: %v", entry.Name(), err)
		}

		if _, err := tx.Exec(context.Background(),
			"INSERT INTO schema_migrations (filename) VALUES ($1)", entry.Name(),
		); err != nil {
			_ = tx.Rollback(context.Background())
			log.Fatalf("failed to record migration %s: %v", entry.Name(), err)
		}

		if err := tx.Commit(context.Background()); err != nil {
			log.Fatalf("failed to commit migration %s: %v", entry.Name(), err)
		}

		log.Printf("applied migration: %s", entry.Name())
	}

	log.Println("all migrations applied successfully")
}
