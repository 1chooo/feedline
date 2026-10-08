package repository

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
	"github.com/1chooo/ad-service/internal/service"
)

// The same contract runs against SQLite on every test run and PostgreSQL when
// TEST_POSTGRES_URL names an isolated test database. It never clears tables.
func TestRepositoryProviders(t *testing.T) {
	configs := []database.Config{{Driver: database.SQLite, SQLitePath: filepath.Join(t.TempDir(), "stream.db")}}
	if connectionURL := os.Getenv("TEST_POSTGRES_URL"); connectionURL != "" {
		configs = append(configs, database.Config{Driver: database.Postgres, URL: connectionURL})
	}
	for _, config := range configs {
		t.Run(string(config.Driver), func(t *testing.T) {
			if config.Driver == database.Postgres {
				config = isolatedPostgresConfig(t, config)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			db, err := database.Open(ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			for i := 0; i < 2; i++ {
				if err := Migrate(ctx, db); err != nil {
					t.Fatal(err)
				}
			}

			t.Run("reproducible fixtures", func(t *testing.T) { testSeedRerun(t, ctx, db) })
			t.Run("auth content media and credits", func(t *testing.T) { testPlatformWorkflow(t, ctx, db) })
			t.Run("concurrent funding", func(t *testing.T) { testConcurrentFunding(t, ctx, db, config) })
			t.Run("UTC analytics", func(t *testing.T) { testUTCAnalytics(t, ctx, db) })
		})
	}
}

func isolatedPostgresConfig(t *testing.T, config database.Config) database.Config {
	t.Helper()
	db, err := database.Open(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("provider_test_%d", time.Now().UnixNano())
	if _, err := db.Exec(context.Background(), "CREATE SCHEMA "+schema); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	// Only this generated test schema is removed; the supplied database's
	// existing application tables and records are never touched.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	connectionURL, err := url.Parse(config.URL)
	if err != nil {
		t.Fatal(err)
	}
	query := connectionURL.Query()
	query.Set("search_path", schema)
	connectionURL.RawQuery = query.Encode()
	config.URL = connectionURL.String()
	return config
}

func testSeedRerun(t *testing.T, ctx context.Context, db database.DB) {
	t.Helper()
	if err := Seed(ctx, db); err != nil {
		t.Fatal(err)
	}
	counts := func() [8]int64 {
		var value [8]int64
		if err := db.QueryRow(ctx, `SELECT
			(SELECT COUNT(*) FROM users), (SELECT COUNT(*) FROM posts),
			(SELECT COUNT(*) FROM companies), (SELECT COUNT(*) FROM ads),
			(SELECT COUNT(*) FROM credit_purchases), (SELECT COUNT(*) FROM credit_transactions),
			(SELECT COUNT(*) FROM ad_events), (SELECT COUNT(*) FROM user_activity)
		`).Scan(&value[0], &value[1], &value[2], &value[3], &value[4], &value[5], &value[6], &value[7]); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := counts()
	if err := Seed(ctx, db); err != nil {
		t.Fatal(err)
	}
	if after := counts(); after != before {
		t.Fatalf("seed added duplicates: before=%v after=%v", before, after)
	}
	if before[0] < 14 || before[2] != 3 || before[4] != 3 {
		t.Fatalf("incomplete fixtures: %v", before)
	}
	var discrepancies int64
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM companies c WHERE c.credit_balance <> (SELECT COALESCE(SUM(delta_credits), 0) FROM credit_transactions WHERE company_id = c.id)`).Scan(&discrepancies); err != nil {
		t.Fatal(err)
	}
	if discrepancies != 0 {
		t.Fatalf("%d company balances do not reconcile", discrepancies)
	}
}

func testPlatformWorkflow(t *testing.T, ctx context.Context, db database.DB) {
	t.Helper()
	users := NewUserRepository(db)
	posts := NewPostRepository(db)
	social := service.NewSocialService(users, posts)
	adminAuth, err := social.Login(ctx, model.LoginRequest{Email: "admin@stream.local", Password: SeedPassword})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := social.RequireAdmin(ctx, adminAuth.Token)
	if err != nil || admin.Role != model.RoleAdmin {
		t.Fatalf("admin session: %v %v", admin, err)
	}
	memberAuth, err := social.Login(ctx, model.LoginRequest{Email: "miles@stream.local", Password: SeedPassword})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := social.RequireAdmin(ctx, memberAuth.Token); err == nil {
		t.Fatal("member gained admin access")
	}
	newMember, err := social.Register(ctx, model.RegisterRequest{Username: "providernew", Email: "new@provider.local", Password: SeedPassword, DisplayName: "Provider Member"})
	if err != nil {
		t.Fatal(err)
	}
	if member, err := social.Me(ctx, newMember.Token); err != nil || member.Role != model.RoleMember {
		t.Fatalf("new registration session: %v %v", member, err)
	}
	if _, err := social.Register(ctx, model.RegisterRequest{Username: "miles", Email: "duplicate@stream.local", Password: SeedPassword, DisplayName: "Duplicate"}); err == nil {
		t.Fatal("duplicate username was accepted")
	} else {
		var conflict *model.ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("duplicate should return a conflict, got %v", err)
		}
	}
	post, err := social.CreatePost(ctx, memberAuth.Token, model.CreatePostRequest{Title: "Provider contract post"})
	if err != nil || post.ID == "" || post.CreatedAt.IsZero() {
		t.Fatalf("create post: %v %v", post, err)
	}
	feed, err := posts.ListPosts(ctx)
	if err != nil || len(feed) < 1 {
		t.Fatalf("read posts: %v %v", feed, err)
	}

	advertiserAuth, err := social.Login(ctx, model.LoginRequest{Email: "kai@stream.local", Password: SeedPassword})
	if err != nil {
		t.Fatal(err)
	}
	advertiser, err := social.RequireAdvertiser(ctx, advertiserAuth.Token)
	if err != nil {
		t.Fatal(err)
	}
	billing := NewBillingRepository(db)
	company, err := billing.EnsureCompany(ctx, advertiser.ID, "Harbor Roast")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := billing.RenameCompany(ctx, advertiser.ID, "Harbor Roast Test"); err != nil {
		t.Fatal(err)
	}
	overview, err := billing.Overview(ctx, advertiser.ID)
	if err != nil || len(overview.Packages) != 3 {
		t.Fatalf("billing overview: %v %v", overview, err)
	}
	initial := overview.Company.CreditBalance
	purchased, err := billing.CompleteManualPurchase(ctx, advertiser.ID, overview.Packages[0].ID, "FALL20", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if purchased.Purchase.AmountCents != 4000 || purchased.Purchase.DiscountCents != 1000 || purchased.Transaction.DeltaCredits != 500 {
		t.Fatalf("incorrect purchase: %+v", purchased)
	}
	if _, err := billing.CompleteManualPurchase(ctx, advertiser.ID, overview.Packages[0].ID, "FALL20", time.Now().UTC()); err == nil {
		t.Fatal("purchase coupon redeemed twice")
	}
	ledger, _, err := billing.RedeemPromoCode(ctx, advertiser.ID, "WELCOME250", time.Now().UTC())
	if err != nil || ledger.DeltaCredits != 250 {
		t.Fatalf("redeem promo: %v %v", ledger, err)
	}
	if _, _, err := billing.RedeemPromoCode(ctx, advertiser.ID, "WELCOME250", time.Now().UTC()); err == nil {
		t.Fatal("bonus code redeemed twice")
	}

	mediaRepo := NewMediaRepository(db)
	image := &model.Media{OwnerID: advertiser.ID, StorageKey: "contract/image.png", URL: "https://example.com/image.png", ContentType: "image/png", SizeBytes: 8}
	if err := mediaRepo.Create(ctx, image); err != nil {
		t.Fatal(err)
	}
	social.WithMediaService(service.NewMediaService(mediaRepo, nil))
	imagePost, err := social.CreatePost(ctx, advertiserAuth.Token, model.CreatePostRequest{Title: "Owned image post", ImageMediaID: &image.ID, ImageUrl: "https://untrusted.example/replacement.png"})
	if err != nil || imagePost.ImageUrl != image.URL {
		t.Fatalf("post did not resolve owned image: %v %v", imagePost, err)
	}
	if _, err := social.CreatePost(ctx, memberAuth.Token, model.CreatePostRequest{Title: "Foreign image post", ImageMediaID: &image.ID}); err == nil {
		t.Fatal("another user could publish owned media")
	}
	objectRemoved := false
	err = mediaRepo.DeleteOwned(ctx, image.ID, advertiser.ID, func(string) error { objectRemoved = true; return nil })
	if err == nil || objectRemoved {
		t.Fatalf("referenced post image was removed: %v", err)
	}
	memberPosts, err := posts.ListPostsByUsername(ctx, advertiser.Username)
	if err != nil {
		t.Fatal(err)
	}
	foundImage := false
	for _, post := range memberPosts {
		if post.ImageMediaID != nil && *post.ImageMediaID == image.ID {
			foundImage = true
		}
	}
	if !foundImage {
		t.Fatal("image reference did not survive post retrieval")
	}
	if foreign, err := mediaRepo.GetByIDAndOwner(ctx, image.ID, admin.ID); err != nil || foreign != nil {
		t.Fatalf("cross-owner image exposed: %v %v", foreign, err)
	}
	ads := NewAdRepository(db)
	ad := &model.Ad{AdvertiserID: &advertiser.ID, ImageMediaID: &image.ID, Title: "Provider contract campaign", Status: model.StatusPaused, StartAt: time.Now().UTC().Add(-time.Hour), EndAt: time.Now().UTC().Add(24 * time.Hour), Conditions: model.Conditions{Country: []string{"US"}}}
	if err := ads.Create(ctx, ad); err != nil {
		t.Fatal(err)
	}
	if canDelete, err := mediaRepo.CanDeleteByIDAndOwner(ctx, image.ID, advertiser.ID); err != nil || canDelete {
		t.Fatalf("campaign creative delete was allowed: %v", err)
	}
	if _, err := billing.FundCampaign(ctx, advertiser.ID, ad.ID, initial+10000, time.Now().UTC()); err == nil {
		t.Fatal("overspend accepted")
	}
	if _, err := billing.FundCampaign(ctx, advertiser.ID, ad.ID, 200, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := billing.FundCampaign(ctx, advertiser.ID, ad.ID, 200, time.Now().UTC()); err == nil {
		t.Fatal("campaign funded twice")
	}
	overview, err = billing.Overview(ctx, advertiser.ID)
	if err != nil || overview.Company.CreditBalance != initial+550 {
		t.Fatalf("balance failed to reconcile: %v %v", overview, err)
	}
	if len(overview.Transactions) != 5 || len(overview.Purchases) != 2 {
		t.Fatalf("unexpected billing history: %+v", overview)
	}
	operations := NewAdminOperationsRepository(db)
	if _, err := operations.UpdateCampaignStatus(ctx, ad.ID, model.StatusCanceled); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.ListCampaigns(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.ListCompanies(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.UpdateUserRole(ctx, advertiser.ID, model.RoleAdvertiser); err != nil {
		t.Fatal(err)
	}
	if _, err := billing.AdjustCredits(ctx, company.ID, 25, "Provider adjustment verification", admin.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	secondAdjustment, err := billing.AdjustCredits(ctx, company.ID, -5, "Second adjustment by the same administrator", admin.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if secondAdjustment.BalanceAfter != initial+570 || secondAdjustment.CreatedByID == nil || *secondAdjustment.CreatedByID != admin.ID {
		t.Fatalf("repeated adjustment lost balance or audit metadata: %+v", secondAdjustment)
	}
	if _, err := billing.ListPromotions(ctx); err != nil {
		t.Fatal(err)
	}
	promotion, err := billing.CreatePromotion(ctx, model.Promotion{Code: "PROVIDER25", Name: "Provider event", Kind: model.PromotionKindEvent, RewardType: model.PromotionRewardBonusCredits, RewardValue: 25, Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := billing.CreatePromotion(ctx, *promotion); err == nil {
		t.Fatal("duplicate promotion accepted")
	} else {
		var conflict *model.ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("duplicate promotion should return a conflict: %v", err)
		}
	}
	if _, err := billing.SetPromotionActive(ctx, promotion.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := billing.RedeemPromoCode(ctx, advertiser.ID, promotion.Code, time.Now().UTC()); err == nil {
		t.Fatal("inactive promotion redeemed")
	}
	spare := &model.Media{OwnerID: advertiser.ID, StorageKey: "contract/spare.png", URL: "https://example.com/spare.png", ContentType: "image/png", SizeBytes: 8}
	if err := mediaRepo.Create(ctx, spare); err != nil {
		t.Fatal(err)
	}
	if err := mediaRepo.DeleteOwned(ctx, spare.ID, advertiser.ID, func(string) error { return errors.New("storage unavailable") }); err == nil {
		t.Fatal("storage failure was not reported")
	}
	if present, err := mediaRepo.GetByIDAndOwner(ctx, spare.ID, advertiser.ID); err != nil || present == nil {
		t.Fatalf("failed deletion lost retryable metadata: %v", err)
	}
	if deleted, err := mediaRepo.DeleteByIDAndOwner(ctx, spare.ID, advertiser.ID); err != nil || !deleted {
		t.Fatalf("delete spare image: %v", err)
	}
	if missing, err := mediaRepo.GetByIDAndOwner(ctx, spare.ID, advertiser.ID); err != nil || missing != nil {
		t.Fatalf("deleted media still present: %v %v", missing, err)
	}
}

func testConcurrentFunding(t *testing.T, ctx context.Context, db database.DB, config database.Config) {
	t.Helper()
	user := &model.User{Username: "race", Email: "race@stream.local", PasswordHash: "not-used", DisplayName: "Concurrent advertiser", Role: model.RoleAdvertiser}
	if err := NewUserRepository(db).CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	billing := NewBillingRepository(db)
	company, err := billing.EnsureCompany(ctx, user.ID, "Race Company")
	if err != nil {
		t.Fatal(err)
	}
	packages, err := billing.ActivePackages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := billing.CompleteManualPurchase(ctx, user.ID, packages[0].ID, "", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	ads := NewAdRepository(db)
	campaigns := make([]model.Ad, 2)
	for i := range campaigns {
		campaigns[i] = model.Ad{AdvertiserID: &user.ID, Title: "Concurrent funding", Status: model.StatusPaused, StartAt: time.Now().UTC().Add(-time.Hour), EndAt: time.Now().UTC().Add(time.Hour)}
		if err := ads.Create(ctx, &campaigns[i]); err != nil {
			t.Fatal(err)
		}
	}
	// A second SQLite connection pool verifies database locking across pools,
	// beyond the local one-connection setting. PostgreSQL retains row locks.
	second, err := database.Open(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	stores := []*BillingRepository{billing, NewBillingRepository(second)}
	results := make(chan error, 2)
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := range campaigns {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			_, err := stores[i].FundCampaign(ctx, user.ID, campaigns[i].ID, 400, time.Now().UTC())
			results <- err
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			var conflict *model.ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("expected insufficient funds, got %v", err)
			}
		}
	}
	if successes != 1 {
		t.Fatalf("%d funding attempts succeeded; want one", successes)
	}
	var balance, delta int64
	if err := db.QueryRow(ctx, `SELECT credit_balance, (SELECT SUM(delta_credits) FROM credit_transactions WHERE company_id = $1) FROM companies WHERE id = $1`, company.ID).Scan(&balance, &delta); err != nil {
		t.Fatal(err)
	}
	if balance != 100 || delta != 100 {
		t.Fatalf("concurrent balance=%d ledger=%d; want 100", balance, delta)
	}
}

func testUTCAnalytics(t *testing.T, ctx context.Context, db database.DB) {
	t.Helper()
	users := NewUserRepository(db)
	user, err := users.GetByUsername(ctx, "jane")
	if err != nil {
		t.Fatal(err)
	}
	ads := NewAdRepository(db)
	campaigns, err := ads.ListByAdvertiser(ctx, user.ID)
	if err != nil || len(campaigns) == 0 {
		t.Fatalf("seed campaigns: %v", err)
	}
	day := time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := ads.InsertAdEvents(ctx, []model.AdEvent{
		{AdID: campaigns[0].ID, EventType: model.AdEventImpression, OccurredAt: day.Add(-time.Nanosecond)},
		{AdID: campaigns[0].ID, EventType: model.AdEventImpression, OccurredAt: day},
		{AdID: campaigns[0].ID, EventType: model.AdEventClick, OccurredAt: day.Add(12 * time.Hour)},
		{AdID: campaigns[0].ID, EventType: model.AdEventImpression, OccurredAt: day.Add(24 * time.Hour)},
	}); err != nil {
		t.Fatal(err)
	}
	summary, err := ads.AnalyticsForAdvertiser(ctx, user.ID, day, day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Impressions != 1 || summary.Clicks != 1 || len(summary.Daily) != 1 || summary.Daily[0].Date != "2030-01-02" {
		t.Fatalf("UTC range or day boundaries changed: %+v", summary)
	}
	now := time.Now().UTC()
	if err := users.RecordActivity(ctx, user.ID, "session", now); err != nil {
		t.Fatal(err)
	}
	if err := users.RecordActivity(ctx, user.ID, "session", now); err != nil {
		t.Fatal(err)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	admin, err := NewAdminAnalyticsRepository(db).Summary(ctx, today.AddDate(0, 0, -29), today.AddDate(0, 0, 1), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(admin.Daily) != 30 || admin.Users.DAU < 1 || admin.Users.MAU < admin.Users.DAU || admin.Billing.RevenueCents < 52000 {
		t.Fatalf("incomplete persisted analytics: %+v", admin)
	}
	if admin.Billing.CreditsUsed != 2900 {
		t.Fatalf("campaign use includes unrelated adjustments: %d", admin.Billing.CreditsUsed)
	}
	// After moving all activity to a distant range, an empty chart should still
	// contain the requested days, with no invented delivery or revenue.
	empty, err := NewAdminAnalyticsRepository(db).Summary(ctx, day.AddDate(0, 0, 10), day.AddDate(0, 0, 13), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Daily) != 3 || empty.Advertising.Impressions != 0 || empty.Billing.RevenueCents != 0 {
		t.Fatalf("empty range: %+v", empty)
	}
}
