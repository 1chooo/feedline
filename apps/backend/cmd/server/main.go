package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/1chooo/ad-service/internal/database"
	httpdelivery "github.com/1chooo/ad-service/internal/delivery/http"
	"github.com/1chooo/ad-service/internal/repository"
	"github.com/1chooo/ad-service/internal/service"
	"github.com/1chooo/ad-service/internal/storage"
)

func main() {
	addr := ":" + envOrDefault("PORT", "8080")
	ctx := context.Background()
	db, err := database.Open(ctx, database.ConfigFromEnv())
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer db.Close()
	if err := repository.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	log.Printf("database provider: %s", db.Dialect())
	repo := repository.NewAdRepository(db)

	events := service.NewBufferedEventSink(repo, 10000, time.Second)
	defer events.Close()
	svc := service.NewAdService(repo).WithEventSink(events)
	social := service.NewSocialService(
		repository.NewUserRepository(db),
		repository.NewPostRepository(db),
	)
	objectStorage, err := storage.New(ctx, storage.ConfigFromEnv())
	if err != nil {
		log.Fatalf("initialize media storage: %v", err)
	}
	media := service.NewMediaService(repository.NewMediaRepository(db), objectStorage)
	analytics := service.NewAnalyticsService(repo)
	adminAnalytics := service.NewAdminAnalyticsService(repository.NewAdminAnalyticsRepository(db))
	adminOperations := service.NewAdminOperationsService(repository.NewAdminOperationsRepository(db))
	billing := service.NewBillingService(
		repository.NewBillingRepository(db),
		envOrDefault("PAYMENTS_DRIVER", "manual"),
		envOrDefault("APP_ENV", "development"),
	)

	if err := svc.RefreshCache(ctx); err != nil {
		log.Fatalf("warm cache: %v", err)
	}

	userCount, err := repository.CountUsers(ctx, db)
	if err != nil {
		log.Printf("count users: %v", err)
	} else if userCount == 0 {
		log.Printf("database has no users; run go run ./cmd/seed with the same database configuration")
	}

	refreshCtx, stopRefresh := context.WithCancel(context.Background())
	defer stopRefresh()
	go runCacheRefresher(refreshCtx, svc)

	handler := httpdelivery.NewHandler(svc, social, media, analytics, adminAnalytics, adminOperations, billing, objectStorage)
	server := &http.Server{
		Addr:              addr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("ad-service listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}

func runCacheRefresher(ctx context.Context, svc *service.AdService) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.RefreshCache(ctx); err != nil {
				log.Printf("refresh cache: %v", err)
			}
		}
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
