package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort  string
	DatabaseURL string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	MaxUploadSize int64

	RateLimitAuth    int
	RateLimitGeneral int

	MaxShareTTL               time.Duration
	RecycleBinExpiry          time.Duration
	RecycleBinCleanupInterval time.Duration

	AdminEmail    string
	AdminPassword string

	CORSAllowedOrigins []string

	LogLevel string
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:                getEnv("SERVER_PORT", "8080"),
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		MinIOEndpoint:             getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:            os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:            os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:               getEnv("MINIO_BUCKET", "nuonetdisk"),
		MinIOUseSSL:               getEnvBool("MINIO_USE_SSL", false),
		JWTSecret:                 os.Getenv("JWT_SECRET"),
		JWTAccessExpiry:           getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry:          getEnvDuration("JWT_REFRESH_EXPIRY", 720*time.Hour),
		MaxUploadSize:             getEnvInt64("MAX_UPLOAD_SIZE", 100*1024*1024),
		RateLimitAuth:             getEnvInt("RATE_LIMIT_AUTH", 10),
		RateLimitGeneral:          getEnvInt("RATE_LIMIT_GENERAL", 100),
		AdminEmail:                os.Getenv("ADMIN_EMAIL"),
		AdminPassword:             getEnv("ADMIN_PASSWORD", "admin123"),
		MaxShareTTL:               getEnvDuration("MAX_SHARE_TTL", 168*time.Hour),
		RecycleBinExpiry:          getEnvDuration("RECYCLE_BIN_EXPIRY", 720*time.Hour),
		RecycleBinCleanupInterval: getEnvDuration("RECYCLE_BIN_CLEANUP_INTERVAL", 24*time.Hour),
		CORSAllowedOrigins:        getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		LogLevel:                  getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.MinIOAccessKey == "" {
		return fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if c.MinIOSecretKey == "" {
		return fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.MaxUploadSize <= 0 {
		return fmt.Errorf("MAX_UPLOAD_SIZE must be positive")
	}
	if c.MaxShareTTL <= 0 {
		return fmt.Errorf("MAX_SHARE_TTL must be positive")
	}
	if c.RateLimitAuth <= 0 {
		return fmt.Errorf("RATE_LIMIT_AUTH must be positive")
	}
	if c.RateLimitGeneral <= 0 {
		return fmt.Errorf("RATE_LIMIT_GENERAL must be positive")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func getEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return i
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}

func getEnvSlice(key string, defaultVal []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parts := strings.Split(val, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
