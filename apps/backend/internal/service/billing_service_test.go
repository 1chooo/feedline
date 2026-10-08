package service

import (
	"context"
	"testing"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type mockBillingStore struct {
	company       *model.Company
	purchaseOwner int64
	purchaseID    int64
	promoCode     string
	fundOwner     int64
	fundCampaign  int64
	fundCredits   int64
}

func (m *mockBillingStore) EnsureCompany(_ context.Context, _ int64, _ string) (*model.Company, error) {
	return m.company, nil
}

func (m *mockBillingStore) RenameCompany(_ context.Context, _ int64, _ string) (*model.Company, error) {
	return m.company, nil
}

func (m *mockBillingStore) Overview(_ context.Context, _ int64) (*model.BillingOverview, error) {
	return &model.BillingOverview{}, nil
}

func (m *mockBillingStore) ActivePackages(_ context.Context) ([]model.CreditPackage, error) {
	return []model.CreditPackage{}, nil
}

func (m *mockBillingStore) CompleteManualPurchase(_ context.Context, ownerID, packageID int64, promoCode string, _ time.Time) (*model.PurchaseCreditsResponse, error) {
	m.purchaseOwner = ownerID
	m.purchaseID = packageID
	m.promoCode = promoCode
	return &model.PurchaseCreditsResponse{}, nil
}

func (m *mockBillingStore) RedeemPromoCode(_ context.Context, _ int64, _ string, _ time.Time) (*model.CreditTransaction, *model.Promotion, error) {
	return &model.CreditTransaction{}, &model.Promotion{}, nil
}

func (m *mockBillingStore) AdjustCredits(_ context.Context, _ int64, _ int64, _ string, _ int64, _ time.Time) (*model.CreditTransaction, error) {
	return &model.CreditTransaction{}, nil
}

func (m *mockBillingStore) FundCampaign(_ context.Context, ownerID, campaignID, credits int64, _ time.Time) (*model.CampaignFundingResponse, error) {
	m.fundOwner = ownerID
	m.fundCampaign = campaignID
	m.fundCredits = credits
	return &model.CampaignFundingResponse{}, nil
}

func (m *mockBillingStore) ListPromotions(_ context.Context) ([]model.Promotion, error) {
	return []model.Promotion{}, nil
}

func (m *mockBillingStore) CreatePromotion(_ context.Context, promotion model.Promotion) (*model.Promotion, error) {
	return &promotion, nil
}

func (m *mockBillingStore) SetPromotionActive(_ context.Context, id int64, active bool) (*model.Promotion, error) {
	return &model.Promotion{ID: id, Active: active}, nil
}

func TestBillingServicePurchaseCreditsUsesManualDriverInDevelopment(t *testing.T) {
	store := &mockBillingStore{}
	svc := NewBillingService(store, "manual", "development")
	svc.WithClock(func() time.Time { return time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC) })

	_, err := svc.PurchaseCredits(context.Background(), 42, model.PurchaseCreditsRequest{PackageID: 7, PromoCode: " welcome20 "})
	if err != nil {
		t.Fatalf("PurchaseCredits() error = %v", err)
	}
	if store.purchaseOwner != 42 || store.purchaseID != 7 || store.promoCode != "WELCOME20" {
		t.Fatalf("manual purchase arguments = (%d, %d, %q)", store.purchaseOwner, store.purchaseID, store.promoCode)
	}
}

func TestBillingServicePurchaseCreditsRejectsManualDriverInProduction(t *testing.T) {
	svc := NewBillingService(&mockBillingStore{}, "manual", "production")
	_, err := svc.PurchaseCredits(context.Background(), 42, model.PurchaseCreditsRequest{PackageID: 7})
	if err == nil {
		t.Fatal("PurchaseCredits() error = nil, want production payment configuration error")
	}
}

func TestBillingServicePurchaseCreditsValidatesPackage(t *testing.T) {
	svc := NewBillingService(&mockBillingStore{}, "manual", "development")
	_, err := svc.PurchaseCredits(context.Background(), 42, model.PurchaseCreditsRequest{})
	if err == nil {
		t.Fatal("PurchaseCredits() error = nil, want validation error")
	}
}

func TestBillingServiceFundsCampaignWithPositiveCredits(t *testing.T) {
	store := &mockBillingStore{}
	svc := NewBillingService(store, "manual", "development")
	_, err := svc.FundCampaign(context.Background(), 8, 21, model.FundCampaignRequest{Credits: 500})
	if err != nil {
		t.Fatalf("FundCampaign() error = %v", err)
	}
	if store.fundOwner != 8 || store.fundCampaign != 21 || store.fundCredits != 500 {
		t.Fatalf("fund arguments = (%d, %d, %d)", store.fundOwner, store.fundCampaign, store.fundCredits)
	}
}

func TestBillingServiceRejectsEmptyCampaignFunding(t *testing.T) {
	svc := NewBillingService(&mockBillingStore{}, "manual", "development")
	_, err := svc.FundCampaign(context.Background(), 8, 21, model.FundCampaignRequest{})
	if err == nil {
		t.Fatal("FundCampaign() error = nil, want validation error")
	}
}
