package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

// SeedPassword is deliberately only for local development and test fixtures.
// The seed command refuses production by default; see cmd/seed for its guard.
const SeedPassword = "password123"

type seedUser struct {
	Username       string
	Email          string
	DisplayName    string
	Bio            string
	Age            int
	Gender         string
	Country        string
	Role           string
	CreatedDaysAgo int
}

type seedPost struct {
	Username       string
	Title          string
	Description    string
	ImageURL       string
	LandingPageURL string
	DaysAgo        int
}

type seedCampaign struct {
	Advertiser       string
	Company          string
	Title            string
	Description      string
	ImageURL         string
	LandingPageURL   string
	Bid              float64
	DailyBudget      int64
	CreditBudget     int64
	CreditSpent      int64
	Status           string
	StartDaysFromNow int
	EndDaysFromNow   int
	Conditions       model.Conditions
	Impressions      int
}

// Seed uses stable accounts, provider references, and record identity checks so
// it is safe to run repeatedly. It never truncates tables or deletes local
// developer data.
func Seed(ctx context.Context, db database.DB) error {
	clock := time.Now().UTC()
	// A day-level anchor keeps generated activity, campaign dates, and ad events
	// stable across reruns on the same day while still making fixtures feel current.
	now := time.Date(clock.Year(), clock.Month(), clock.Day(), 12, 0, 0, 0, time.UTC)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	users := seedUsers()
	userIDs := make(map[string]int64, len(users))
	for _, user := range users {
		id, err := upsertSeedUser(ctx, db, user, string(passwordHash), now.AddDate(0, 0, -user.CreatedDaysAgo))
		if err != nil {
			return err
		}
		userIDs[user.Username] = id
	}
	if err := seedUserActivity(ctx, db, userIDs, users, now); err != nil {
		return err
	}
	if err := seedPosts(ctx, db, userIDs, now); err != nil {
		return err
	}
	if err := seedCreditPackages(ctx, db); err != nil {
		return err
	}
	if err := seedPromotions(ctx, db, now); err != nil {
		return err
	}

	companyIDs := map[string]int64{}
	for username, name := range map[string]string{
		"jane": "Northline Outdoor",
		"kai":  "Harbor Roast",
		"nova": "Lumen Labs",
	} {
		companyID, err := ensureSeedCompany(ctx, db, userIDs[username], name)
		if err != nil {
			return err
		}
		companyIDs[username] = companyID
	}

	if err := seedBillingHistory(ctx, db, companyIDs, now); err != nil {
		return err
	}
	if err := seedCampaignsAndEvents(ctx, db, userIDs, companyIDs, now); err != nil {
		return err
	}
	return nil
}

func upsertSeedUser(ctx context.Context, db database.DB, user seedUser, passwordHash string, createdAt time.Time) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, bio, age, gender, country, role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (email) DO UPDATE SET
			username = EXCLUDED.username,
			password_hash = EXCLUDED.password_hash,
			display_name = EXCLUDED.display_name,
			bio = EXCLUDED.bio,
			age = EXCLUDED.age,
			gender = EXCLUDED.gender,
			country = EXCLUDED.country,
			role = EXCLUDED.role,
			created_at = EXCLUDED.created_at
		RETURNING id
	`, user.Username, user.Email, passwordHash, user.DisplayName, user.Bio, user.Age, user.Gender, user.Country, user.Role, createdAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert seed user %s: %w", user.Username, err)
	}
	return id, nil
}

func seedUserActivity(ctx context.Context, db database.DB, userIDs map[string]int64, users []seedUser, now time.Time) error {
	for username, userID := range userIDs {
		for daysAgo := 0; daysAgo < 45; daysAgo++ {
			if (daysAgo+len(username))%4 == 0 {
				continue
			}
			occurredAt := now.AddDate(0, 0, -daysAgo).Add(time.Duration((daysAgo+len(username))%10) * time.Hour)
			if _, err := db.Exec(ctx, `
				INSERT INTO user_activity (user_id, activity_type, activity_date, occurred_at)
				VALUES ($1, 'session', $2, $3)
				ON CONFLICT (user_id, activity_type, activity_date) DO NOTHING
			`, userID, occurredAt.UTC().Format(time.DateOnly), occurredAt); err != nil {
				return fmt.Errorf("seed activity for %s: %w", username, err)
			}
		}
	}
	for _, user := range users {
		userID := userIDs[user.Username]
		registeredAt := now.AddDate(0, 0, -user.CreatedDaysAgo)
		for _, day := range []int{1, 7, 30} {
			if user.CreatedDaysAgo <= day {
				continue
			}
			occurredAt := registeredAt.AddDate(0, 0, day).Add(2 * time.Hour)
			if _, err := db.Exec(ctx, `
				INSERT INTO user_activity (user_id, activity_type, activity_date, occurred_at)
				VALUES ($1, 'session', $2, $3)
				ON CONFLICT (user_id, activity_type, activity_date) DO NOTHING
			`, userID, occurredAt.UTC().Format(time.DateOnly), occurredAt); err != nil {
				return fmt.Errorf("seed retention activity for %s: %w", user.Username, err)
			}
		}
	}
	return nil
}

func seedPosts(ctx context.Context, db database.DB, userIDs map[string]int64, now time.Time) error {
	for _, post := range seedPostsData() {
		ownerID := userIDs[post.Username]
		if ownerID == 0 {
			return fmt.Errorf("seed post owner %s does not exist", post.Username)
		}
		createdAt := now.AddDate(0, 0, -post.DaysAgo)
		if _, err := db.Exec(ctx, `
			INSERT INTO posts (user_id, title, description, image_url, landing_page_url, created_at)
			SELECT $1, $2, $3, $4, $5, $6
			WHERE NOT EXISTS (SELECT 1 FROM posts WHERE user_id = $1 AND title = $2)
		`, ownerID, post.Title, post.Description, post.ImageURL, post.LandingPageURL, createdAt); err != nil {
			return fmt.Errorf("seed post %q: %w", post.Title, err)
		}
	}
	return nil
}

func seedCreditPackages(ctx context.Context, db database.DB) error {
	packages := []model.CreditPackage{
		{Code: "STARTER", Name: "Starter", PriceCents: 5000, Currency: "USD", Credits: 500, BonusCredits: 0, Active: true},
		{Code: "GROWTH", Name: "Growth", PriceCents: 15000, Currency: "USD", Credits: 1800, BonusCredits: 200, Active: true},
		{Code: "SCALE", Name: "Scale", PriceCents: 40000, Currency: "USD", Credits: 5600, BonusCredits: 900, Active: true},
	}
	for _, creditPackage := range packages {
		if _, err := db.Exec(ctx, `
			INSERT INTO credit_packages (code, name, price_cents, currency, credits, bonus_credits, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (code) DO UPDATE SET
				name = EXCLUDED.name, price_cents = EXCLUDED.price_cents, currency = EXCLUDED.currency,
				credits = EXCLUDED.credits, bonus_credits = EXCLUDED.bonus_credits, active = EXCLUDED.active
		`, creditPackage.Code, creditPackage.Name, creditPackage.PriceCents, creditPackage.Currency, creditPackage.Credits, creditPackage.BonusCredits, creditPackage.Active); err != nil {
			return fmt.Errorf("seed credit package %s: %w", creditPackage.Code, err)
		}
	}
	return nil
}

func seedPromotions(ctx context.Context, db database.DB, now time.Time) error {
	startsAt := now.AddDate(0, 0, -30)
	endsAt := now.AddDate(0, 1, 0)
	promotions := []model.Promotion{
		{Code: "WELCOME250", Name: "Welcome credit", Kind: model.PromotionKindCoupon, RewardType: model.PromotionRewardBonusCredits, RewardValue: 250, StartsAt: &startsAt, EndsAt: &endsAt, MaxRedemptions: int64Ptr(500), Active: true},
		{Code: "FALL20", Name: "Seasonal 20% discount", Kind: model.PromotionKindCoupon, RewardType: model.PromotionRewardPercentDiscount, RewardValue: 20, StartsAt: &startsAt, EndsAt: &endsAt, MaxRedemptions: int64Ptr(500), Active: true},
		{Code: "LAUNCH500", Name: "Launch event credit", Kind: model.PromotionKindEvent, RewardType: model.PromotionRewardBonusCredits, RewardValue: 500, StartsAt: &startsAt, EndsAt: &endsAt, Active: true},
	}
	for _, promotion := range promotions {
		if _, err := db.Exec(ctx, `
			INSERT INTO promotions (code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (code) DO UPDATE SET
				name = EXCLUDED.name, kind = EXCLUDED.kind, reward_type = EXCLUDED.reward_type, reward_value = EXCLUDED.reward_value,
				starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at, max_redemptions = EXCLUDED.max_redemptions, active = EXCLUDED.active
		`, promotion.Code, promotion.Name, promotion.Kind, promotion.RewardType, promotion.RewardValue, promotion.StartsAt, promotion.EndsAt, promotion.MaxRedemptions, promotion.Active); err != nil {
			return fmt.Errorf("seed promotion %s: %w", promotion.Code, err)
		}
	}
	return nil
}

func ensureSeedCompany(ctx context.Context, db database.DB, ownerID int64, name string) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO companies (owner_id, name)
		VALUES ($1, $2)
		ON CONFLICT (owner_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, ownerID, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("seed company %s: %w", name, err)
	}
	return id, nil
}

func seedBillingHistory(ctx context.Context, db database.DB, companyIDs map[string]int64, now time.Time) error {
	if err := seedPurchase(ctx, db, companyIDs["jane"], "GROWTH", "seed:northline:growth", 0, "", now.AddDate(0, 0, -21)); err != nil {
		return err
	}
	if err := seedPurchase(ctx, db, companyIDs["kai"], "STARTER", "seed:harbor:starter", 0, "", now.AddDate(0, 0, -12)); err != nil {
		return err
	}
	if err := seedPurchase(ctx, db, companyIDs["nova"], "SCALE", "seed:lumen:scale", 8000, "FALL20", now.AddDate(0, 0, -6)); err != nil {
		return err
	}
	if err := seedPromotionCredit(ctx, db, companyIDs["jane"], "WELCOME250", "seed:northline:welcome", now.AddDate(0, 0, -20)); err != nil {
		return err
	}
	if err := seedPromotionCredit(ctx, db, companyIDs["nova"], "LAUNCH500", "seed:lumen:launch", now.AddDate(0, 0, -5)); err != nil {
		return err
	}
	return nil
}

func seedPurchase(ctx context.Context, db database.DB, companyID int64, packageCode, providerReference string, discountCents int64, promotionCode string, createdAt time.Time) error {
	var creditPackage model.CreditPackage
	if err := db.QueryRow(ctx, `SELECT id, code, name, price_cents, currency, credits, bonus_credits, active, created_at FROM credit_packages WHERE code = $1`, packageCode).Scan(&creditPackage.ID, &creditPackage.Code, &creditPackage.Name, &creditPackage.PriceCents, &creditPackage.Currency, &creditPackage.Credits, &creditPackage.BonusCredits, &creditPackage.Active, &creditPackage.CreatedAt); err != nil {
		return fmt.Errorf("load seed credit package %s: %w", packageCode, err)
	}
	var purchaseID int64
	err := db.QueryRow(ctx, `
		INSERT INTO credit_purchases (company_id, package_id, status, provider, provider_reference, amount_cents, discount_cents, currency, credits, created_at)
		VALUES ($1, $2, 'completed', 'seed', $3, $4, $5, $6, $7, $8)
		ON CONFLICT (provider, provider_reference) DO UPDATE SET provider_reference = credit_purchases.provider_reference
		RETURNING id
	`, companyID, creditPackage.ID, providerReference, creditPackage.PriceCents-discountCents, discountCents, creditPackage.Currency, creditPackage.Credits+creditPackage.BonusCredits, createdAt).Scan(&purchaseID)
	if err != nil {
		return fmt.Errorf("seed purchase %s: %w", providerReference, err)
	}
	if err := seedCreditTransaction(ctx, db, companyID, model.CreditTransactionPurchase, creditPackage.Credits+creditPackage.BonusCredits, "seed:purchase:"+providerReference, "Seed credit package purchase", createdAt); err != nil {
		return err
	}
	if promotionCode != "" {
		var redemptionID int64
		err := db.QueryRow(ctx, `
			INSERT INTO promotion_redemptions (promotion_id, company_id, purchase_id, created_at)
			SELECT id, $1, $2, $3 FROM promotions WHERE code = $4
			ON CONFLICT (promotion_id, company_id) DO NOTHING
			RETURNING id
		`, companyID, purchaseID, createdAt, promotionCode).Scan(&redemptionID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("seed purchase promotion %s: %w", promotionCode, err)
		}
		if _, err := db.Exec(ctx, `
			UPDATE promotions SET total_redemptions = total_redemptions + 1
			WHERE code = $1
		`, promotionCode); err != nil {
			return fmt.Errorf("update seed purchase promotion count %s: %w", promotionCode, err)
		}
	}
	return nil
}

func seedPromotionCredit(ctx context.Context, db database.DB, companyID int64, code, reference string, createdAt time.Time) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin seed promotion credit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var promotionID, credits int64
	var name string
	if err := tx.QueryRow(ctx, `SELECT id, name, reward_value FROM promotions WHERE code = $1`, code).Scan(&promotionID, &name, &credits); err != nil {
		return fmt.Errorf("load seed promotion %s: %w", code, err)
	}
	var redemptionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO promotion_redemptions (promotion_id, company_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (promotion_id, company_id) DO NOTHING
		RETURNING id
	`, promotionID, companyID, createdAt).Scan(&redemptionID)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("seed promotion redemption: %w", err)
	}
	if err := seedCreditTransactionTx(ctx, tx, companyID, model.CreditTransactionPromotion, credits, reference, name, createdAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE promotions SET total_redemptions = total_redemptions + 1 WHERE id = $1`, promotionID); err != nil {
		return fmt.Errorf("update seed promotion count: %w", err)
	}
	return tx.Commit(ctx)
}

func seedCreditTransaction(ctx context.Context, db database.DB, companyID int64, transactionType string, delta int64, reference, note string, createdAt time.Time) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin seed credit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := seedCreditTransactionTx(ctx, tx, companyID, transactionType, delta, reference, note, createdAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func seedCreditTransactionTx(ctx context.Context, tx database.Tx, companyID int64, transactionType string, delta int64, reference, note string, createdAt time.Time) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM credit_transactions WHERE company_id = $1 AND reference = $2)`, companyID, reference).Scan(&exists); err != nil {
		return fmt.Errorf("check seed credit transaction: %w", err)
	}
	if exists {
		return nil
	}
	var balance int64
	if err := tx.QueryRow(ctx, `SELECT credit_balance FROM companies WHERE id = $1`+database.ForUpdate(tx), companyID).Scan(&balance); err != nil {
		return fmt.Errorf("lock seed company credits: %w", err)
	}
	if balance+delta < 0 {
		return fmt.Errorf("seed transaction %s would make company balance negative", reference)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO credit_transactions (company_id, type, delta_credits, balance_after, reference, note, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, companyID, transactionType, delta, balance+delta, reference, note, createdAt); err != nil {
		return fmt.Errorf("insert seed credit transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE companies SET credit_balance = $1 WHERE id = $2`, balance+delta, companyID); err != nil {
		return fmt.Errorf("update seed company balance: %w", err)
	}
	return nil
}

func seedCampaignsAndEvents(ctx context.Context, db database.DB, userIDs, companyIDs map[string]int64, now time.Time) error {
	for _, campaign := range seedCampaignData() {
		adID, err := seedCampaignRecord(ctx, db, campaign, userIDs[campaign.Advertiser], companyIDs[campaign.Company], now)
		if err != nil {
			return err
		}
		if campaign.CreditSpent > 0 {
			if err := seedCreditTransaction(ctx, db, companyIDs[campaign.Company], model.CreditTransactionCampaignSpend, -campaign.CreditSpent, "seed:campaign:"+campaign.Title, "Seed campaign delivery", now.AddDate(0, 0, -1)); err != nil {
				return err
			}
		}
		if err := seedAdEvents(ctx, db, adID, campaign.Impressions, now); err != nil {
			return err
		}
	}
	return nil
}

func seedCampaignRecord(ctx context.Context, db database.DB, campaign seedCampaign, advertiserID, companyID int64, now time.Time) (int64, error) {
	if advertiserID == 0 || companyID == 0 {
		return 0, fmt.Errorf("campaign %q has an unknown advertiser or company", campaign.Title)
	}
	conditions, err := json.Marshal(campaign.Conditions)
	if err != nil {
		return 0, fmt.Errorf("marshal campaign conditions: %w", err)
	}
	startAt := now.AddDate(0, 0, campaign.StartDaysFromNow)
	endAt := now.AddDate(0, 0, campaign.EndDaysFromNow)
	var id int64
	err = db.QueryRow(ctx, `
		INSERT INTO ads (advertiser_id, company_id, title, description, image_url, landing_page_url, bid, daily_budget, credit_budget, credit_spent, status, start_at, end_at, conditions)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		WHERE NOT EXISTS (SELECT 1 FROM ads WHERE advertiser_id = $1 AND title = $3)
		RETURNING id
	`, advertiserID, companyID, campaign.Title, campaign.Description, campaign.ImageURL, campaign.LandingPageURL, campaign.Bid, campaign.DailyBudget, campaign.CreditBudget, campaign.CreditSpent, campaign.Status, startAt, endAt, conditions).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.QueryRow(ctx, `SELECT id FROM ads WHERE advertiser_id = $1 AND title = $2`, advertiserID, campaign.Title).Scan(&id)
	}
	if err != nil {
		return 0, fmt.Errorf("seed campaign %q: %w", campaign.Title, err)
	}
	return id, nil
}

func seedAdEvents(ctx context.Context, db database.DB, adID int64, impressions int, now time.Time) error {
	for i := 0; i < impressions; i++ {
		eventType := model.AdEventImpression
		if i%19 == 0 {
			eventType = model.AdEventClick
		}
		occurredAt := now.Add(-time.Duration(i*3+1) * time.Hour).Truncate(time.Second)
		if _, err := db.Exec(ctx, `
			INSERT INTO ad_events (ad_id, event_type, occurred_at)
			SELECT $1, $2, $3
			WHERE NOT EXISTS (SELECT 1 FROM ad_events WHERE ad_id = $1 AND event_type = $2 AND occurred_at = $3)
		`, adID, eventType, occurredAt); err != nil {
			return fmt.Errorf("seed ad event: %w", err)
		}
	}
	return nil
}

func seedUsers() []seedUser {
	return []seedUser{
		{Username: "admin", Email: "admin@stream.local", DisplayName: "Avery Admin", Bio: "Platform operations and advertiser success.", Age: 37, Gender: model.GenderFemale, Country: "US", Role: model.RoleAdmin, CreatedDaysAgo: 180},
		{Username: "ops", Email: "ops@stream.local", DisplayName: "Jordan Ops", Bio: "Trust, safety, and platform operations.", Age: 32, Gender: model.GenderMale, Country: "US", Role: model.RoleAdmin, CreatedDaysAgo: 150},
		{Username: "jane", Email: "jane@stream.local", DisplayName: "Jane Park", Bio: "Northline Outdoor. Gear for slow weekends outside.", Age: 28, Gender: model.GenderFemale, Country: "US", Role: model.RoleAdvertiser, CreatedDaysAgo: 120},
		{Username: "kai", Email: "kai@stream.local", DisplayName: "Kai Rivera", Bio: "Harbor Roast. Small-batch coffee from the waterfront.", Age: 34, Gender: model.GenderMale, Country: "US", Role: model.RoleAdvertiser, CreatedDaysAgo: 80},
		{Username: "nova", Email: "nova@stream.local", DisplayName: "Nova Chen", Bio: "Lumen Labs. Quiet tech for busy days.", Age: 26, Gender: model.GenderFemale, Country: "TW", Role: model.RoleAdvertiser, CreatedDaysAgo: 65},
		{Username: "miles", Email: "miles@stream.local", DisplayName: "Miles Ortega", Bio: "Notes on walking, cities, and ordinary ads that feel human.", Age: 41, Gender: model.GenderMale, Country: "US", Role: model.RoleMember, CreatedDaysAgo: 50},
		{Username: "rhea", Email: "rhea@stream.local", DisplayName: "Rhea Singh", Bio: "Sketching interface ideas over lunch.", Age: 29, Gender: model.GenderFemale, Country: "IN", Role: model.RoleMember, CreatedDaysAgo: 45},
		{Username: "sam", Email: "sam@stream.local", DisplayName: "Sam Lee", Bio: "Cycling routes and neighborhood coffee.", Age: 31, Gender: model.GenderMale, Country: "CA", Role: model.RoleMember, CreatedDaysAgo: 40},
		{Username: "inez", Email: "inez@stream.local", DisplayName: "Inez Moreno", Bio: "Photos from the 6:15 train.", Age: 24, Gender: model.GenderFemale, Country: "ES", Role: model.RoleMember, CreatedDaysAgo: 33},
		{Username: "dev", Email: "dev@stream.local", DisplayName: "Dev Shah", Bio: "Making tiny games after work.", Age: 27, Gender: model.GenderMale, Country: "GB", Role: model.RoleMember, CreatedDaysAgo: 25},
		{Username: "mina", Email: "mina@stream.local", DisplayName: "Mina Kato", Bio: "Ceramics, late breakfasts, and thrifted books.", Age: 35, Gender: model.GenderFemale, Country: "JP", Role: model.RoleMember, CreatedDaysAgo: 18},
		{Username: "noah", Email: "noah@stream.local", DisplayName: "Noah Bell", Bio: "Trying one new recipe every week.", Age: 38, Gender: model.GenderMale, Country: "AU", Role: model.RoleMember, CreatedDaysAgo: 12},
		{Username: "ada", Email: "ada@stream.local", DisplayName: "Ada Brooks", Bio: "Museum notes and long walks.", Age: 30, Gender: model.GenderFemale, Country: "US", Role: model.RoleMember, CreatedDaysAgo: 7},
		{Username: "omar", Email: "omar@stream.local", DisplayName: "Omar Haddad", Bio: "Music, maps, and market mornings.", Age: 33, Gender: model.GenderMale, Country: "AE", Role: model.RoleMember, CreatedDaysAgo: 3},
	}
}

func seedPostsData() []seedPost {
	return []seedPost{
		{Username: "jane", Title: "The pack that disappears on the trail", Description: "12 liters, no bounce, and a bottle pocket you can reach with one hand.", ImageURL: "https://picsum.photos/id/1015/800/800", LandingPageURL: "https://example.com/northline-pack", DaysAgo: 2},
		{Username: "kai", Title: "Harbor blend, roasted this morning", Description: "Chocolate, orange peel, and a finish that stays with the walk home.", ImageURL: "https://picsum.photos/id/1080/800/800", LandingPageURL: "https://example.com/harbor-roast", DaysAgo: 5},
		{Username: "nova", Title: "Headphones that stay out of the way", Description: "Soft clamp, 30-hour battery, and a case that fits a jacket pocket.", ImageURL: "https://picsum.photos/id/180/800/800", LandingPageURL: "https://example.com/lumen-quiet", DaysAgo: 8},
		{Username: "miles", Title: "Why I still walk to work", Description: "Twenty minutes each way. No playlist. Just the same block of shops.", LandingPageURL: "https://example.com/miles-walk", DaysAgo: 11},
		{Username: "rhea", Title: "Tiny rituals for a calmer inbox", Description: "Three labels, one unsubscribe pass, and a Friday reset.", ImageURL: "https://picsum.photos/id/1062/800/800", DaysAgo: 14},
		{Username: "sam", Title: "The route with the best sunrise", Description: "A six-mile loop with no cars for the first twenty minutes.", ImageURL: "https://picsum.photos/id/1036/800/800", DaysAgo: 17},
		{Username: "inez", Title: "Rain on platform four", Description: "The city looks softer through a train window.", ImageURL: "https://picsum.photos/id/1040/800/800", DaysAgo: 20},
		{Username: "dev", Title: "A game about packing lunch", Description: "It is surprisingly hard to balance the perfect sandwich.", DaysAgo: 23},
		{Username: "mina", Title: "A bowl that took three tries", Description: "The glaze finally came out the color of moss.", ImageURL: "https://picsum.photos/id/1025/800/800", DaysAgo: 26},
		{Username: "noah", Title: "Weeknight noodles, no recipe", Description: "Garlic, greens, an egg, and whatever sauce is open.", DaysAgo: 29},
		{Username: "ada", Title: "An hour with one painting", Description: "A quiet gallery is the best kind of reset.", ImageURL: "https://picsum.photos/id/1039/800/800", DaysAgo: 32},
		{Username: "omar", Title: "Market morning", Description: "Fresh dates, mint tea, and a bag too full to carry well.", ImageURL: "https://picsum.photos/id/1067/800/800", DaysAgo: 35},
	}
}

func seedCampaignData() []seedCampaign {
	web := []string{model.PlatformWeb}
	return []seedCampaign{
		{Advertiser: "jane", Company: "jane", Title: "Northline trail season", Description: "Waterproof layers that pack into their own pocket.", ImageURL: "https://picsum.photos/id/1018/800/800", LandingPageURL: "https://example.com/trail-season", Bid: 1.25, DailyBudget: 600, CreditBudget: 1200, CreditSpent: 650, Status: model.StatusActive, StartDaysFromNow: -14, EndDaysFromNow: 21, Conditions: model.Conditions{Platform: web}, Impressions: 320},
		{Advertiser: "jane", Company: "jane", Title: "Northline winter preview", Description: "The lightest layer for early cold-weather weekends.", ImageURL: "https://picsum.photos/id/1016/800/800", LandingPageURL: "https://example.com/winter-preview", Bid: 1.1, DailyBudget: 400, CreditBudget: 550, CreditSpent: 0, Status: model.StatusActive, StartDaysFromNow: 8, EndDaysFromNow: 40, Conditions: model.Conditions{Country: []string{"US"}, Platform: web}, Impressions: 0},
		{Advertiser: "kai", Company: "kai", Title: "Harbor pour-over kit", Description: "A kettle, a dripper, and beans roasted this week.", ImageURL: "https://picsum.photos/id/1060/800/800", LandingPageURL: "https://example.com/pour-over", Bid: 2.5, DailyBudget: 450, CreditBudget: 800, CreditSpent: 350, Status: model.StatusActive, StartDaysFromNow: -9, EndDaysFromNow: 16, Conditions: model.Conditions{Country: []string{"US", "CA"}, Platform: web}, Impressions: 250},
		{Advertiser: "nova", Company: "nova", Title: "Lumen quiet launch", Description: "Soft clamp, focused sound, and thirty quiet hours.", ImageURL: "https://picsum.photos/id/160/800/800", LandingPageURL: "https://example.com/lumen-launch", Bid: 2.0, DailyBudget: 700, CreditBudget: 2200, CreditSpent: 1300, Status: model.StatusArchived, StartDaysFromNow: -40, EndDaysFromNow: -4, Conditions: model.Conditions{Country: []string{"TW", "JP"}, Platform: web}, Impressions: 480},
		{Advertiser: "nova", Company: "nova", Title: "Lumen canceled experiment", Description: "An unpublished targeting experiment.", LandingPageURL: "https://example.com/lumen-experiment", Bid: 0.8, DailyBudget: 100, CreditBudget: 100, CreditSpent: 0, Status: model.StatusCanceled, StartDaysFromNow: -2, EndDaysFromNow: 8, Conditions: model.Conditions{Platform: web}, Impressions: 0},
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
