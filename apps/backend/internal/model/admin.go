package model

import (
	"strings"
	"time"
)

type AdminUser struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"createdAt"`
}

type AdminCompany struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	CreditBalance int64     `json:"creditBalance"`
	OwnerUsername string    `json:"ownerUsername"`
	OwnerEmail    string    `json:"ownerEmail"`
	CreatedAt     time.Time `json:"createdAt"`
}

type AdminCampaign struct {
	ID                 int64     `json:"id"`
	Title              string    `json:"title"`
	Status             string    `json:"status"`
	AdvertiserUsername string    `json:"advertiserUsername,omitempty"`
	CompanyName        string    `json:"companyName,omitempty"`
	CreditBudget       *int64    `json:"creditBudget,omitempty"`
	CreditSpent        int64     `json:"creditSpent"`
	StartAt            time.Time `json:"startAt"`
	EndAt              time.Time `json:"endAt"`
	CreatedAt          time.Time `json:"createdAt"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

type UpdateCampaignStatusRequest struct {
	Status string `json:"status"`
}

type AdminPromotionRequest struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	RewardType     string `json:"rewardType"`
	RewardValue    int64  `json:"rewardValue"`
	StartsAt       string `json:"startsAt,omitempty"`
	EndsAt         string `json:"endsAt,omitempty"`
	MaxRedemptions *int64 `json:"maxRedemptions,omitempty"`
	Active         *bool  `json:"active,omitempty"`
}

type AdminPromotionStatusRequest struct {
	Active bool `json:"active"`
}

func ValidateManagedRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case RoleMember, RoleAdvertiser, RoleAdmin:
		return role, nil
	default:
		return "", invalid("role must be member, advertiser, or admin")
	}
}

func ValidateManagedCampaignStatus(status string) (string, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case StatusActive, StatusPaused, StatusArchived, StatusCanceled:
		return status, nil
	default:
		return "", invalid("status must be active, paused, archived, or canceled")
	}
}

func ValidateAdminPromotion(req AdminPromotionRequest) (Promotion, error) {
	promotion := Promotion{
		Code:           strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:           strings.TrimSpace(req.Name),
		Kind:           strings.ToLower(strings.TrimSpace(req.Kind)),
		RewardType:     strings.ToLower(strings.TrimSpace(req.RewardType)),
		RewardValue:    req.RewardValue,
		MaxRedemptions: req.MaxRedemptions,
		Active:         true,
	}
	if req.Active != nil {
		promotion.Active = *req.Active
	}
	if promotion.Code == "" || len(promotion.Code) > 64 {
		return Promotion{}, invalid("promo code must be between 1 and 64 characters")
	}
	if promotion.Name == "" || len(promotion.Name) > 120 {
		return Promotion{}, invalid("promotion name must be between 1 and 120 characters")
	}
	if promotion.Kind != PromotionKindCoupon && promotion.Kind != PromotionKindEvent && promotion.Kind != PromotionKindPurchase {
		return Promotion{}, invalid("promotion kind must be coupon, event, or purchase")
	}
	if promotion.RewardType != PromotionRewardBonusCredits && promotion.RewardType != PromotionRewardPercentDiscount {
		return Promotion{}, invalid("rewardType must be bonus_credits or percent_discount")
	}
	if promotion.RewardValue < 1 || (promotion.RewardType == PromotionRewardPercentDiscount && promotion.RewardValue > 100) {
		return Promotion{}, invalid("rewardValue is invalid for the selected reward type")
	}
	if promotion.MaxRedemptions != nil && *promotion.MaxRedemptions < 1 {
		return Promotion{}, invalid("maxRedemptions must be a positive integer")
	}
	var err error
	if req.StartsAt != "" {
		value, parseErr := time.Parse(time.RFC3339, req.StartsAt)
		if parseErr != nil {
			return Promotion{}, invalid("startsAt must be an ISO 8601 timestamp")
		}
		promotion.StartsAt = &value
	}
	if req.EndsAt != "" {
		value, parseErr := time.Parse(time.RFC3339, req.EndsAt)
		if parseErr != nil {
			return Promotion{}, invalid("endsAt must be an ISO 8601 timestamp")
		}
		promotion.EndsAt = &value
	}
	if promotion.StartsAt != nil && promotion.EndsAt != nil && !promotion.EndsAt.After(*promotion.StartsAt) {
		err = invalid("endsAt must be after startsAt")
	}
	if err != nil {
		return Promotion{}, err
	}
	return promotion, nil
}
