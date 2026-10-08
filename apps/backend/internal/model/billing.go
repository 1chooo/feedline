package model

import (
	"strings"
	"time"
)

const (
	CreditTransactionPurchase        = "purchase"
	CreditTransactionPromotion       = "promotion"
	CreditTransactionAdminAdjustment = "admin_adjustment"
	CreditTransactionCampaignSpend   = "campaign_spend"
	CreditTransactionCampaignRefund  = "campaign_refund"
	PurchaseStatusCompleted          = "completed"
	PromotionKindCoupon              = "coupon"
	PromotionKindEvent               = "event"
	PromotionKindPurchase            = "purchase"
	PromotionRewardBonusCredits      = "bonus_credits"
	PromotionRewardPercentDiscount   = "percent_discount"
)

// Company is the billing owner for an advertiser. Each advertiser starts with
// one company, but the schema leaves room for future team and multi-company
// support without coupling money to an individual user account.
type Company struct {
	ID            int64     `json:"id"`
	OwnerID       int64     `json:"ownerId"`
	Name          string    `json:"name"`
	CreditBalance int64     `json:"creditBalance"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CreditPackage struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	PriceCents   int64     `json:"priceCents"`
	Currency     string    `json:"currency"`
	Credits      int64     `json:"credits"`
	BonusCredits int64     `json:"bonusCredits"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Promotion struct {
	ID               int64      `json:"id"`
	Code             string     `json:"code,omitempty"`
	Name             string     `json:"name"`
	Kind             string     `json:"kind"`
	RewardType       string     `json:"rewardType"`
	RewardValue      int64      `json:"rewardValue"`
	StartsAt         *time.Time `json:"startsAt,omitempty"`
	EndsAt           *time.Time `json:"endsAt,omitempty"`
	MaxRedemptions   *int64     `json:"maxRedemptions,omitempty"`
	TotalRedemptions int64      `json:"totalRedemptions"`
	Active           bool       `json:"active"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type CreditTransaction struct {
	ID           int64     `json:"id"`
	CompanyID    int64     `json:"companyId"`
	Type         string    `json:"type"`
	DeltaCredits int64     `json:"deltaCredits"`
	BalanceAfter int64     `json:"balanceAfter"`
	Reference    string    `json:"reference,omitempty"`
	Note         string    `json:"note,omitempty"`
	CreatedByID  *int64    `json:"createdById,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CreditPurchase struct {
	ID                int64     `json:"id"`
	CompanyID         int64     `json:"companyId"`
	PackageID         int64     `json:"packageId"`
	Status            string    `json:"status"`
	Provider          string    `json:"provider"`
	ProviderReference string    `json:"providerReference"`
	AmountCents       int64     `json:"amountCents"`
	DiscountCents     int64     `json:"discountCents"`
	Currency          string    `json:"currency"`
	Credits           int64     `json:"credits"`
	CreatedAt         time.Time `json:"createdAt"`
}

type BillingOverview struct {
	Checkout     CheckoutAvailability `json:"checkout"`
	Company      *Company             `json:"company,omitempty"`
	Packages     []CreditPackage      `json:"packages"`
	Transactions []CreditTransaction  `json:"transactions"`
	Purchases    []CreditPurchase     `json:"purchases"`
}

type CheckoutAvailability struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
	Message string `json:"message"`
}

type CreateCompanyRequest struct {
	Name string `json:"name"`
}

type PurchaseCreditsRequest struct {
	PackageID int64  `json:"packageId"`
	PromoCode string `json:"promoCode,omitempty"`
}

type RedeemPromoCodeRequest struct {
	Code string `json:"code"`
}

type AdminCreditAdjustmentRequest struct {
	CompanyID    int64  `json:"companyId"`
	DeltaCredits int64  `json:"deltaCredits"`
	Note         string `json:"note"`
}

type FundCampaignRequest struct {
	Credits int64 `json:"credits"`
}

type CampaignFundingResponse struct {
	Campaign    Ad                `json:"campaign"`
	Transaction CreditTransaction `json:"transaction"`
}

type PurchaseCreditsResponse struct {
	Purchase    CreditPurchase    `json:"purchase"`
	Transaction CreditTransaction `json:"transaction"`
	Promotion   *Promotion        `json:"promotion,omitempty"`
}

func ValidateCompanyName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("company name is required")
	}
	if len(name) > 120 {
		return "", invalid("company name must be at most 120 characters")
	}
	return name, nil
}

func ValidatePurchaseCreditsRequest(req PurchaseCreditsRequest) (string, error) {
	if req.PackageID < 1 {
		return "", invalid("packageId must be a positive integer")
	}
	code := strings.ToUpper(strings.TrimSpace(req.PromoCode))
	if len(code) > 64 {
		return "", invalid("promoCode must be at most 64 characters")
	}
	return code, nil
}

func ValidatePromoCode(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return "", invalid("promo code is required")
	}
	if len(code) > 64 {
		return "", invalid("promo code must be at most 64 characters")
	}
	return code, nil
}

func ValidateCreditAdjustment(req AdminCreditAdjustmentRequest) (string, error) {
	if req.CompanyID < 1 {
		return "", invalid("companyId must be a positive integer")
	}
	if req.DeltaCredits == 0 {
		return "", invalid("deltaCredits must not be zero")
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return "", invalid("note is required for a credit adjustment")
	}
	if len(note) > 280 {
		return "", invalid("note must be at most 280 characters")
	}
	return note, nil
}

func ValidateCampaignFunding(req FundCampaignRequest) error {
	if req.Credits < 1 {
		return invalid("credits must be a positive integer")
	}
	return nil
}
