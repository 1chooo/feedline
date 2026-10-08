package main

import (
	"context"
	"log"
	"os"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/repository"
)

func main() {
	if os.Getenv("APP_ENV") == "production" && os.Getenv("ALLOW_PRODUCTION_SEED") != "true" {
		log.Fatal("refusing to seed production; use isolated development/test data instead")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.ConfigFromEnv())
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := repository.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := repository.Seed(ctx, db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("seed complete (%s)", db.Dialect())
}
