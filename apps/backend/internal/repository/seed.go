package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const SeedPassword = "password123"

type seedUser struct {
	Username    string
	Email       string
	DisplayName string
	Bio         string
	Age         int
	Gender      string
	Country     string
	Posts       []seedPost
}

type seedPost struct {
	Title          string
	Description    string
	ImageUrl       string
	LandingPageUrl string
}

type seedAd struct {
	Title          string
	Description    string
	ImageUrl       string
	LandingPageUrl string
	Bid            float64
	Status         string
	Conditions     model.Conditions
}

func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	if err := seedUsersAndPosts(ctx, pool); err != nil {
		return err
	}
	return seedAds(ctx, pool)
}

func seedUsersAndPosts(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	for _, user := range seedUsers() {
		var userID int64
		err := pool.QueryRow(ctx, `
			INSERT INTO users (username, email, password_hash, display_name, bio, age, gender, country)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`, user.Username, user.Email, string(passwordHash), user.DisplayName, user.Bio, user.Age, user.Gender, user.Country).Scan(&userID)
		if err != nil {
			return fmt.Errorf("seed user %s: %w", user.Username, err)
		}

		for _, post := range user.Posts {
			if _, err := pool.Exec(ctx, `
				INSERT INTO posts (user_id, title, description, image_url, landing_page_url)
				VALUES ($1, $2, $3, $4, $5)
			`, userID, post.Title, post.Description, post.ImageUrl, post.LandingPageUrl); err != nil {
				return fmt.Errorf("seed post for %s: %w", user.Username, err)
			}
		}
	}

	return nil
}

func seedAds(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ads`).Scan(&count); err != nil {
		return fmt.Errorf("count ads: %w", err)
	}
	if count > 0 {
		return nil
	}

	startAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC)

	for _, ad := range seedAdsData() {
		conditionsJSON, err := json.Marshal(ad.Conditions)
		if err != nil {
			return fmt.Errorf("marshal seed ad conditions: %w", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO ads (title, description, image_url, landing_page_url, bid, status, start_at, end_at, conditions)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, ad.Title, ad.Description, ad.ImageUrl, ad.LandingPageUrl, ad.Bid, ad.Status, startAt, endAt, conditionsJSON); err != nil {
			return fmt.Errorf("seed ad %s: %w", ad.Title, err)
		}
	}

	return nil
}

func seedUsers() []seedUser {
	return []seedUser{
		{
			Username:    "jane",
			Email:       "jane@stream.local",
			DisplayName: "Jane Park",
			Bio:         "Northline Outdoor. Gear for slow weekends outside.",
			Age:         28,
			Gender:      model.GenderFemale,
			Country:     "US",
			Posts: []seedPost{
				{
					Title:          "The pack that disappears on the trail",
					Description:    "12 liters, no bounce, and a bottle pocket you can reach with one hand.",
					ImageUrl:       "https://picsum.photos/id/1015/800/800",
					LandingPageUrl: "https://example.com/northline-pack",
				},
				{
					Title:          "Weekend tent for two",
					Description:    "Sets up in the time it takes to boil water. Packs smaller than a loaf of bread.",
					ImageUrl:       "https://picsum.photos/id/1016/800/800",
					LandingPageUrl: "https://example.com/northline-tent",
				},
			},
		},
		{
			Username:    "kai",
			Email:       "kai@stream.local",
			DisplayName: "Kai Rivera",
			Bio:         "Harbor Roast. Small-batch coffee from the waterfront.",
			Age:         34,
			Gender:      model.GenderMale,
			Country:     "US",
			Posts: []seedPost{
				{
					Title:          "Harbor blend, roasted this morning",
					Description:    "Chocolate, orange peel, and a finish that stays with the walk home.",
					ImageUrl:       "https://picsum.photos/id/1080/800/800",
					LandingPageUrl: "https://example.com/harbor-roast",
				},
			},
		},
		{
			Username:    "nova",
			Email:       "nova@stream.local",
			DisplayName: "Nova Chen",
			Bio:         "Lumen Labs. Quiet tech for busy days.",
			Age:         26,
			Gender:      model.GenderFemale,
			Country:     "TW",
			Posts: []seedPost{
				{
					Title:          "Headphones that stay out of the way",
					Description:    "Soft clamp, 30-hour battery, and a case that fits a jacket pocket.",
					ImageUrl:       "https://picsum.photos/id/180/800/800",
					LandingPageUrl: "https://example.com/lumen-quiet",
				},
			},
		},
		{
			Username:    "miles",
			Email:       "miles@stream.local",
			DisplayName: "Miles Ortega",
			Bio:         "Notes on walking, cities, and ordinary ads that feel human.",
			Age:         41,
			Gender:      model.GenderMale,
			Country:     "US",
			Posts: []seedPost{
				{
					Title:          "Why I still walk to work",
					Description:    "Twenty minutes each way. No playlist. Just the same block of shops and one billboard that finally learned my name.",
					LandingPageUrl: "https://example.com/miles-walk",
				},
			},
		},
	}
}

func seedAdsData() []seedAd {
	web := []string{model.PlatformWeb}
	return []seedAd{
		{
			Title:          "Trail season starts now",
			Description:    "Waterproof layers that pack into their own pocket.",
			ImageUrl:       "https://picsum.photos/id/1018/800/800",
			LandingPageUrl: "https://example.com/trail-season",
			Bid:            1.25,
			Status:         model.StatusActive,
			Conditions:     model.Conditions{Platform: web},
		},
		{
			Title:          "Harbor pour-over kit",
			Description:    "A kettle, a dripper, and beans roasted this week.",
			ImageUrl:       "https://picsum.photos/id/1060/800/800",
			LandingPageUrl: "https://example.com/pour-over",
			Bid:            2.5,
			Status:         model.StatusActive,
			Conditions: model.Conditions{
				AgeStart: intPtr(25),
				AgeEnd:   intPtr(45),
				Country:  []string{"US"},
				Platform: web,
			},
		},
		{
			Title:          "Night shift headphones",
			Description:    "Soft clamp and a case that fits a jacket pocket.",
			ImageUrl:       "https://picsum.photos/id/160/800/800",
			LandingPageUrl: "https://example.com/night-shift",
			Bid:            2.0,
			Status:         model.StatusActive,
			Conditions: model.Conditions{
				Country:  []string{"TW"},
				Platform: web,
			},
		},
		{
			Title:          "City walking club",
			Description:    "Tuesday evenings. Same route. New billboards.",
			ImageUrl:       "https://picsum.photos/id/1019/800/800",
			LandingPageUrl: "https://example.com/walking-club",
			Bid:            1.5,
			Status:         model.StatusActive,
			Conditions:     model.Conditions{Platform: web},
		},
		{
			Title:          "Android-only launch",
			Description:    "This should not appear on the web feed.",
			LandingPageUrl: "https://example.com/android-only",
			Bid:            3.0,
			Status:         model.StatusActive,
			Conditions:     model.Conditions{Platform: []string{model.PlatformAndroid}},
		},
		{
			Title:          "Archived campaign",
			Description:    "Kept in the database so we can see paused history.",
			LandingPageUrl: "https://example.com/archived",
			Bid:            0.5,
			Status:         model.StatusArchived,
			Conditions:     model.Conditions{},
		},
	}
}

func intPtr(v int) *int {
	return &v
}
