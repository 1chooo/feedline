package main

import (
	"context"
	"log"
	"os"

	"github.com/1chooo/ad-service/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := envOrDefault("DATABASE_URL", "postgres://ad:ad@localhost:5432/ad_service?sslmode=disable")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	if err := repository.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := repository.Seed(ctx, pool); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("seed complete")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
