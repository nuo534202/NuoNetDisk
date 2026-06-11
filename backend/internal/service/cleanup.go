package service

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CleanupService struct {
	pool     *pgxpool.Pool
	interval time.Duration
}

func NewCleanupService(pool *pgxpool.Pool, interval time.Duration) *CleanupService {
	return &CleanupService{
		pool:     pool,
		interval: interval,
	}
}

func (s *CleanupService) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("recycle bin cleanup started, interval: %s", s.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("recycle bin cleanup stopped")
			return
		case <-ticker.C:
			s.cleanup(ctx)
		}
	}
}

func (s *CleanupService) cleanup(ctx context.Context) {
	query := `
		DELETE FROM files
		WHERE is_deleted = TRUE
		AND deleted_at < NOW() - INTERVAL '30 days'`

	result, err := s.pool.Exec(ctx, query)
	if err != nil {
		log.Printf("failed to cleanup expired files: %v", err)
		return
	}
	if deleted := result.RowsAffected(); deleted > 0 {
		log.Printf("cleaned up %d expired files", deleted)
	}

	folderQuery := `
		DELETE FROM folders
		WHERE is_deleted = TRUE
		AND deleted_at < NOW() - INTERVAL '30 days'`

	result, err = s.pool.Exec(ctx, folderQuery)
	if err != nil {
		log.Printf("failed to cleanup expired folders: %v", err)
		return
	}
	if deleted := result.RowsAffected(); deleted > 0 {
		log.Printf("cleaned up %d expired folders", deleted)
	}
}
