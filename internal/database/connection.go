package database

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	dns := os.Getenv("DATABASE_URL")
	if dns == "" {
		dns = "postgres://postgres:postgres@localhost:5432/luxsuv-co?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(dns)
	if err != nil {
		return nil, err
	}
	cfg.MinConns = 1
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.HealthCheckPeriod = 30 * time.Second
	return pgxpool.NewWithConfig(ctx, cfg)
}
