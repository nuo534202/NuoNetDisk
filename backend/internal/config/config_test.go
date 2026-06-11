package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidConfig(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("JWT_SECRET", "this-is-a-32-char-secret-key-here!!")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", cfg.DatabaseURL)
	assert.Equal(t, "localhost:9000", cfg.MinIOEndpoint)
	assert.Equal(t, "minioadmin", cfg.MinIOAccessKey)
	assert.Equal(t, "minioadmin", cfg.MinIOSecretKey)
	assert.Equal(t, "nuonetdisk", cfg.MinIOBucket)
	assert.False(t, cfg.MinIOUseSSL)
	assert.Equal(t, 15*time.Minute, cfg.JWTAccessExpiry)
	assert.Equal(t, 720*time.Hour, cfg.JWTRefreshExpiry)
	assert.Equal(t, int64(104857600), cfg.MaxUploadSize)
	assert.Equal(t, 10, cfg.RateLimitAuth)
	assert.Equal(t, 100, cfg.RateLimitGeneral)
	assert.Equal(t, 720*time.Hour, cfg.RecycleBinExpiry)
	assert.Equal(t, 24*time.Hour, cfg.RecycleBinCleanupInterval)
	assert.Equal(t, []string{"*"}, cfg.CORSAllowedOrigins)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	os.Clearenv()
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("JWT_SECRET", "this-is-a-32-char-secret-key-here!!")

	cfg, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
	assert.Nil(t, cfg)
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("JWT_SECRET", "short")

	cfg, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
	assert.Nil(t, cfg)
}

func TestLoad_CustomValues(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("MINIO_ENDPOINT", "minio.example.com:9000")
	os.Setenv("MINIO_ACCESS_KEY", "customkey")
	os.Setenv("MINIO_SECRET_KEY", "customsecret")
	os.Setenv("MINIO_BUCKET", "custom-bucket")
	os.Setenv("MINIO_USE_SSL", "true")
	os.Setenv("JWT_SECRET", "this-is-a-32-char-secret-key-here!!")
	os.Setenv("MAX_UPLOAD_SIZE", "52428800")
	os.Setenv("RATE_LIMIT_AUTH", "20")
	os.Setenv("RATE_LIMIT_GENERAL", "200")
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,https://example.com")
	os.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "minio.example.com:9000", cfg.MinIOEndpoint)
	assert.True(t, cfg.MinIOUseSSL)
	assert.Equal(t, int64(52428800), cfg.MaxUploadSize)
	assert.Equal(t, 20, cfg.RateLimitAuth)
	assert.Equal(t, 200, cfg.RateLimitGeneral)
	assert.Equal(t, []string{"http://localhost:3000", "https://example.com"}, cfg.CORSAllowedOrigins)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestMustLoad_Panics(t *testing.T) {
	os.Clearenv()
	assert.Panics(t, func() {
		MustLoad()
	})
}
