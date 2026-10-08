package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type BillingStore interface {
	EnsureCompany(ctx context.Context, ownerID int64, name string) (*model.Company, error)
	RenameCompany(ctx context.Context, ownerID int64, name string) (*model.Company, error)
	Overview(ctx context.Context, ownerID int64) (*model.BillingOverview, error)
	ActivePackages(ctx context.Context) ([]model.CreditPackage, error)
	CompleteManualPurchase(ctx context.Context, ownerID, packageID int64, promoCode string, now time.Time) (*model.PurchaseCreditsResponse, error)
	RedeemPromoCode(ctx context.Context, ownerID int64, code string, now time.Time) (*model.CreditTransaction, *model.Promotion, error)
	AdjustCredits(ctx context.Context, companyID, delta int64, note string, adminID int64, now time.Time) (*model.CreditTransaction, error)
	FundCampaign(ctx context.Context, ownerID, adID, credits int64, now time.Time) (*model.CampaignFundingResponse, error)
	ListPromotions(ctx context.Context) ([]model.Promotion, error)
	CreatePromotion(ctx context.Context, promotion model.Promotion) (*model.Promotion, error)
	SetPromotionActive(ctx context.Context, id int64, active bool) (*model.Promotion, error)
}

// BillingService defines the payment boundary. The manual driver is deliberately
// development-only: it creates a completed purchase record without pretending
// to charge a card. A production payment adapter can later confirm a provider
// webhook before calling the same ledger operation.
type BillingService struct {
	store          BillingStore
	now            func() time.Time
	paymentDriver  string
	productionMode bool
}

func NewBillingService(store BillingStore, paymentDriver, appEnv string) *BillingService {
	return &BillingService{
		store:          store,
		now:            time.Now,
		paymentDriver:  strings.ToLower(strings.TrimSpace(paymentDriver)),
		productionMode: strings.EqualFold(strings.TrimSpace(appEnv), "production"),
	}
}

func (s *BillingService) WithClock(now func() time.Time) *BillingService {
	s.now = now
	return s
}

func (s *BillingService) EnsureDefaultCompany(ctx context.Context, user *model.User) (*model.Company, error) {
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		name = user.Username
	}
	return s.store.EnsureCompany(ctx, user.ID, name+" Advertising")
}

func (s *BillingService) RenameCompany(ctx context.Context, ownerID int64, req model.CreateCompanyRequest) (*model.Company, error) {
	name, err := model.ValidateCompanyName(req.Name)
	if err != nil {
		return nil, err
	}
	return s.store.RenameCompany(ctx, ownerID, name)
}

func (s *BillingService) Overview(ctx context.Context, ownerID int64) (*model.BillingOverview, error) {
	overview, err := s.store.Overview(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	overview.Checkout = s.CheckoutAvailability()
	return overview, nil
}

func (s *BillingService) CheckoutAvailability() model.CheckoutAvailability {
	if !s.productionMode && s.paymentDriver == "manual" {
		return model.CheckoutAvailability{Enabled: true, Mode: "development", Message: "Development checkout: credits are added for testing. No payment is collected."}
	}
	return model.CheckoutAvailability{Enabled: false, Mode: "unavailable", Message: "Credit checkout is currently unavailable. Contact platform support for credit assistance."}
}

func (s *BillingService) Pricing(ctx context.Context) ([]model.CreditPackage, error) {
	return s.store.ActivePackages(ctx)
}

func (s *BillingService) PurchaseCredits(ctx context.Context, ownerID int64, req model.PurchaseCreditsRequest) (*model.PurchaseCreditsResponse, error) {
	promoCode, err := model.ValidatePurchaseCreditsRequest(req)
	if err != nil {
		return nil, err
	}
	if s.productionMode || s.paymentDriver != "manual" {
		return nil, model.Forbidden("credit checkout is not configured; use the configured payment provider")
	}
	return s.store.CompleteManualPurchase(ctx, ownerID, req.PackageID, promoCode, s.now().UTC())
}

func (s *BillingService) RedeemPromoCode(ctx context.Context, ownerID int64, req model.RedeemPromoCodeRequest) (*model.CreditTransaction, *model.Promotion, error) {
	code, err := model.ValidatePromoCode(req.Code)
	if err != nil {
		return nil, nil, err
	}
	return s.store.RedeemPromoCode(ctx, ownerID, code, s.now().UTC())
}

func (s *BillingService) AdjustCredits(ctx context.Context, adminID int64, req model.AdminCreditAdjustmentRequest) (*model.CreditTransaction, error) {
	note, err := model.ValidateCreditAdjustment(req)
	if err != nil {
		return nil, err
	}
	return s.store.AdjustCredits(ctx, req.CompanyID, req.DeltaCredits, note, adminID, s.now().UTC())
}

func (s *BillingService) FundCampaign(ctx context.Context, ownerID, adID int64, req model.FundCampaignRequest) (*model.CampaignFundingResponse, error) {
	if adID < 1 {
		return nil, model.NotFound("campaign not found")
	}
	if err := model.ValidateCampaignFunding(req); err != nil {
		return nil, err
	}
	return s.store.FundCampaign(ctx, ownerID, adID, req.Credits, s.now().UTC())
}

func (s *BillingService) Promotions(ctx context.Context) ([]model.Promotion, error) {
	return s.store.ListPromotions(ctx)
}

func (s *BillingService) CreatePromotion(ctx context.Context, req model.AdminPromotionRequest) (*model.Promotion, error) {
	promotion, err := model.ValidateAdminPromotion(req)
	if err != nil {
		return nil, err
	}
	return s.store.CreatePromotion(ctx, promotion)
}

func (s *BillingService) SetPromotionActive(ctx context.Context, id int64, req model.AdminPromotionStatusRequest) (*model.Promotion, error) {
	if id < 1 {
		return nil, model.NotFound("promotion not found")
	}
	return s.store.SetPromotionActive(ctx, id, req.Active)
}

func (s *BillingService) RequireManualDriver() error {
	if s.productionMode || s.paymentDriver != "manual" {
		return fmt.Errorf("manual payments are disabled")
	}
	return nil
}
