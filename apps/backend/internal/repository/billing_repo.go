package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

// BillingRepository owns all mutations of a company's credit balance. Balance
// updates and ledger inserts are intentionally performed in one transaction so
// reporting can always reconcile the cached balance to the immutable ledger.
type BillingRepository struct {
	db database.DB
}

func NewBillingRepository(db database.DB) *BillingRepository {
	return &BillingRepository{db: db}
}

func (r *BillingRepository) EnsureCompany(ctx context.Context, ownerID int64, name string) (*model.Company, error) {
	company, err := scanCompany(r.db.QueryRow(ctx, `
		INSERT INTO companies (owner_id, name)
		VALUES ($1, $2)
		ON CONFLICT (owner_id) DO UPDATE SET name = companies.name
		RETURNING id, owner_id, name, credit_balance, created_at
	`, ownerID, name))
	if err != nil {
		return nil, fmt.Errorf("ensure company: %w", err)
	}
	return company, nil
}

func (r *BillingRepository) RenameCompany(ctx context.Context, ownerID int64, name string) (*model.Company, error) {
	company, err := scanCompany(r.db.QueryRow(ctx, `
		UPDATE companies
		SET name = $1
		WHERE owner_id = $2
		RETURNING id, owner_id, name, credit_balance, created_at
	`, name, ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("company not found")
	}
	if err != nil {
		return nil, fmt.Errorf("rename company: %w", err)
	}
	return company, nil
}

func (r *BillingRepository) Overview(ctx context.Context, ownerID int64) (*model.BillingOverview, error) {
	overview := &model.BillingOverview{
		Packages:     []model.CreditPackage{},
		Transactions: []model.CreditTransaction{},
		Purchases:    []model.CreditPurchase{},
	}

	company, err := r.companyByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	overview.Company = company

	packages, err := r.ActivePackages(ctx)
	if err != nil {
		return nil, err
	}
	overview.Packages = packages
	if company == nil {
		return overview, nil
	}

	transactions, err := r.transactionsForCompany(ctx, company.ID, 25)
	if err != nil {
		return nil, err
	}
	overview.Transactions = transactions
	purchases, err := r.purchasesForCompany(ctx, company.ID, 25)
	if err != nil {
		return nil, err
	}
	overview.Purchases = purchases
	return overview, nil
}

func (r *BillingRepository) ActivePackages(ctx context.Context) ([]model.CreditPackage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, price_cents, currency, credits, bonus_credits, active, created_at
		FROM credit_packages
		WHERE active = TRUE
		ORDER BY price_cents ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list credit packages: %w", err)
	}
	defer rows.Close()

	packages := []model.CreditPackage{}
	for rows.Next() {
		creditPackage, err := scanCreditPackage(rows)
		if err != nil {
			return nil, err
		}
		packages = append(packages, *creditPackage)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit packages: %w", err)
	}
	return packages, nil
}

func (r *BillingRepository) ListPromotions(ctx context.Context) ([]model.Promotion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, total_redemptions, active, created_at
		FROM promotions ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list promotions: %w", err)
	}
	defer rows.Close()
	promotions := []model.Promotion{}
	for rows.Next() {
		promotion, err := scanPromotion(rows)
		if err != nil {
			return nil, err
		}
		promotions = append(promotions, *promotion)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate promotions: %w", err)
	}
	return promotions, nil
}

func (r *BillingRepository) CreatePromotion(ctx context.Context, promotion model.Promotion) (*model.Promotion, error) {
	created := &model.Promotion{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO promotions (code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, total_redemptions, active, created_at
	`, promotion.Code, promotion.Name, promotion.Kind, promotion.RewardType, promotion.RewardValue, promotion.StartsAt, promotion.EndsAt, promotion.MaxRedemptions, promotion.Active).Scan(
		&created.ID, &created.Code, &created.Name, &created.Kind, &created.RewardType, &created.RewardValue, &created.StartsAt, &created.EndsAt, &created.MaxRedemptions, &created.TotalRedemptions, &created.Active, &created.CreatedAt,
	)
	if err != nil {
		return nil, mapPromotionWriteError(err)
	}
	return created, nil
}

func (r *BillingRepository) SetPromotionActive(ctx context.Context, id int64, active bool) (*model.Promotion, error) {
	promotion, err := scanPromotion(r.db.QueryRow(ctx, `
		UPDATE promotions SET active = $1 WHERE id = $2
		RETURNING id, code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, total_redemptions, active, created_at
	`, active, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("promotion not found")
	}
	if err != nil {
		return nil, fmt.Errorf("set promotion active: %w", err)
	}
	return promotion, nil
}

func (r *BillingRepository) CompleteManualPurchase(ctx context.Context, ownerID, packageID int64, promoCode string, now time.Time) (*model.PurchaseCreditsResponse, error) {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin credit purchase: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	company, err := companyByOwnerForUpdate(ctx, tx, ownerID)
	if err != nil {
		return nil, err
	}
	creditPackage, err := activePackageForUpdate(ctx, tx, packageID)
	if err != nil {
		return nil, err
	}

	credits := creditPackage.Credits + creditPackage.BonusCredits
	discountCents := int64(0)
	var promotion *model.Promotion
	if promoCode != "" {
		promotion, err = availablePromotionForUpdate(ctx, tx, promoCode, now)
		if err != nil {
			return nil, err
		}
		if promotion.Kind != model.PromotionKindCoupon && promotion.Kind != model.PromotionKindPurchase {
			return nil, model.Forbidden("promo code cannot be used for a purchase")
		}
		if err := assertNotRedeemed(ctx, tx, promotion.ID, company.ID); err != nil {
			return nil, err
		}
		switch promotion.RewardType {
		case model.PromotionRewardBonusCredits:
			credits += promotion.RewardValue
		case model.PromotionRewardPercentDiscount:
			discountCents = creditPackage.PriceCents * promotion.RewardValue / 100
		default:
			return nil, model.NotFound("promo code unavailable")
		}
	}

	purchase := model.CreditPurchase{}
	err = tx.QueryRow(ctx, `
		INSERT INTO credit_purchases (company_id, package_id, status, provider, provider_reference, amount_cents, discount_cents, currency, credits)
		VALUES ($1, $2, $3, 'manual', 'manual', $4, $5, $6, $7)
		RETURNING id, company_id, package_id, status, provider, provider_reference, amount_cents, discount_cents, currency, credits, created_at
	`, company.ID, creditPackage.ID, model.PurchaseStatusCompleted, creditPackage.PriceCents-discountCents, discountCents, creditPackage.Currency, credits).Scan(
		&purchase.ID, &purchase.CompanyID, &purchase.PackageID, &purchase.Status, &purchase.Provider, &purchase.ProviderReference,
		&purchase.AmountCents, &purchase.DiscountCents, &purchase.Currency, &purchase.Credits, &purchase.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create credit purchase: %w", err)
	}
	purchase.ProviderReference = fmt.Sprintf("manual:%d", purchase.ID)
	if _, err := tx.Exec(ctx, `UPDATE credit_purchases SET provider_reference = $1 WHERE id = $2`, purchase.ProviderReference, purchase.ID); err != nil {
		return nil, fmt.Errorf("finalize credit purchase: %w", err)
	}

	ledger, err := appendCredits(ctx, tx, company.ID, model.CreditTransactionPurchase, credits, "purchase:"+fmt.Sprint(purchase.ID), "Credit package purchase", nil, now)
	if err != nil {
		return nil, err
	}
	if promotion != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO promotion_redemptions (promotion_id, company_id, purchase_id)
			VALUES ($1, $2, $3)
		`, promotion.ID, company.ID, purchase.ID); err != nil {
			return nil, fmt.Errorf("record promotion redemption: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE promotions SET total_redemptions = total_redemptions + 1 WHERE id = $1`, promotion.ID); err != nil {
			return nil, fmt.Errorf("increment promotion redemptions: %w", err)
		}
		promotion.TotalRedemptions++
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credit purchase: %w", err)
	}
	return &model.PurchaseCreditsResponse{Purchase: purchase, Transaction: *ledger, Promotion: promotion}, nil
}

func (r *BillingRepository) RedeemPromoCode(ctx context.Context, ownerID int64, code string, now time.Time) (*model.CreditTransaction, *model.Promotion, error) {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin promo redemption: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	company, err := companyByOwnerForUpdate(ctx, tx, ownerID)
	if err != nil {
		return nil, nil, err
	}
	promotion, err := availablePromotionForUpdate(ctx, tx, code, now)
	if err != nil {
		return nil, nil, err
	}
	if promotion.RewardType != model.PromotionRewardBonusCredits {
		return nil, nil, model.Forbidden("promo code must be applied to a credit purchase")
	}
	if err := assertNotRedeemed(ctx, tx, promotion.ID, company.ID); err != nil {
		return nil, nil, err
	}

	ledger, err := appendCredits(ctx, tx, company.ID, model.CreditTransactionPromotion, promotion.RewardValue, "promotion:"+fmt.Sprint(promotion.ID), promotion.Name, nil, now)
	if err != nil {
		return nil, nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO promotion_redemptions (promotion_id, company_id) VALUES ($1, $2)`, promotion.ID, company.ID); err != nil {
		return nil, nil, fmt.Errorf("record promo redemption: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE promotions SET total_redemptions = total_redemptions + 1 WHERE id = $1`, promotion.ID); err != nil {
		return nil, nil, fmt.Errorf("increment promo redemptions: %w", err)
	}
	promotion.TotalRedemptions++
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit promo redemption: %w", err)
	}
	return ledger, promotion, nil
}

func (r *BillingRepository) AdjustCredits(ctx context.Context, companyID, delta int64, note string, adminID int64, now time.Time) (*model.CreditTransaction, error) {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin credit adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedCompanyID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM companies WHERE id = $1`+database.ForUpdate(tx), companyID).Scan(&lockedCompanyID); errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("company not found")
	} else if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}
	// Each adjustment is distinct; the actor belongs in audit metadata rather
	// than a unique reference that would block this admin's later adjustments.
	ledger, err := appendCredits(ctx, tx, companyID, model.CreditTransactionAdminAdjustment, delta, "", note, &adminID, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credit adjustment: %w", err)
	}
	return ledger, nil
}

func (r *BillingRepository) FundCampaign(ctx context.Context, ownerID, adID, credits int64, now time.Time) (*model.CampaignFundingResponse, error) {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin campaign funding: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	company, err := companyByOwnerForUpdate(ctx, tx, ownerID)
	if err != nil {
		return nil, err
	}
	campaign, err := scanAd(tx.QueryRow(ctx, `
		SELECT id, advertiser_id, company_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, credit_budget, credit_spent, status, start_at, end_at, conditions, created_at
		FROM ads
		WHERE id = $1 AND advertiser_id = $2
	`+database.ForUpdate(tx), adID, ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("campaign not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load campaign for funding: %w", err)
	}
	if campaign.Status != model.StatusPaused {
		return nil, model.Conflict("only paused campaigns can be funded")
	}
	if !campaign.EndAt.After(now) {
		return nil, model.Conflict("campaign has already ended")
	}
	if campaign.CreditBudget != nil {
		return nil, model.Conflict("campaign has already been funded")
	}

	ledger, err := appendCredits(ctx, tx, company.ID, model.CreditTransactionCampaignSpend, -credits, "campaign:"+fmt.Sprint(campaign.ID)+":budget", "Campaign credit budget", nil, now)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE ads
		SET company_id = $1, credit_budget = $2, credit_spent = $2, status = 'active'
		WHERE id = $3
	`, company.ID, credits, campaign.ID); err != nil {
		return nil, fmt.Errorf("fund campaign: %w", err)
	}
	campaign.CompanyID = &company.ID
	campaign.CreditBudget = &credits
	campaign.CreditSpent = credits
	campaign.Status = model.StatusActive
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit campaign funding: %w", err)
	}
	return &model.CampaignFundingResponse{Campaign: campaign, Transaction: *ledger}, nil
}

func (r *BillingRepository) companyByOwner(ctx context.Context, ownerID int64) (*model.Company, error) {
	company, err := scanCompany(r.db.QueryRow(ctx, `
		SELECT id, owner_id, name, credit_balance, created_at FROM companies WHERE owner_id = $1
	`, ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}
	return company, nil
}

func (r *BillingRepository) transactionsForCompany(ctx context.Context, companyID int64, limit int) ([]model.CreditTransaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, company_id, type, delta_credits, balance_after, reference, note, created_by_id, created_at
		FROM credit_transactions WHERE company_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2
	`, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("list credit transactions: %w", err)
	}
	defer rows.Close()
	transactions := []model.CreditTransaction{}
	for rows.Next() {
		transaction, err := scanCreditTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit transactions: %w", err)
	}
	return transactions, nil
}

func (r *BillingRepository) purchasesForCompany(ctx context.Context, companyID int64, limit int) ([]model.CreditPurchase, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, company_id, package_id, status, provider, provider_reference, amount_cents, discount_cents, currency, credits, created_at
		FROM credit_purchases WHERE company_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2
	`, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("list credit purchases: %w", err)
	}
	defer rows.Close()
	purchases := []model.CreditPurchase{}
	for rows.Next() {
		purchase, err := scanCreditPurchase(rows)
		if err != nil {
			return nil, err
		}
		purchases = append(purchases, *purchase)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit purchases: %w", err)
	}
	return purchases, nil
}

func companyByOwnerForUpdate(ctx context.Context, tx database.Tx, ownerID int64) (*model.Company, error) {
	company, err := scanCompany(tx.QueryRow(ctx, `
		SELECT id, owner_id, name, credit_balance, created_at FROM companies WHERE owner_id = $1
	`+database.ForUpdate(tx), ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("company not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}
	return company, nil
}

func activePackageForUpdate(ctx context.Context, tx database.Tx, packageID int64) (*model.CreditPackage, error) {
	creditPackage, err := scanCreditPackage(tx.QueryRow(ctx, `
		SELECT id, code, name, price_cents, currency, credits, bonus_credits, active, created_at
		FROM credit_packages WHERE id = $1 AND active = TRUE
	`+database.ForUpdate(tx), packageID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("credit package not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load credit package: %w", err)
	}
	return creditPackage, nil
}

func availablePromotionForUpdate(ctx context.Context, tx database.Tx, code string, now time.Time) (*model.Promotion, error) {
	promotion, err := scanPromotion(tx.QueryRow(ctx, `
		SELECT id, code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, total_redemptions, active, created_at
		FROM promotions
		WHERE code = $1 AND active = TRUE
		  AND (starts_at IS NULL OR starts_at <= $2)
		  AND (ends_at IS NULL OR ends_at > $2)
		  AND (max_redemptions IS NULL OR total_redemptions < max_redemptions)
	`+database.ForUpdate(tx), code, now))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("promo code unavailable")
	}
	if err != nil {
		return nil, fmt.Errorf("load promotion: %w", err)
	}
	return promotion, nil
}

func assertNotRedeemed(ctx context.Context, tx database.Tx, promotionID, companyID int64) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM promotion_redemptions WHERE promotion_id = $1 AND company_id = $2)
	`, promotionID, companyID).Scan(&exists); err != nil {
		return fmt.Errorf("check promotion redemption: %w", err)
	}
	if exists {
		return model.Conflict("promo code has already been redeemed by this company")
	}
	return nil
}

func appendCredits(ctx context.Context, tx database.Tx, companyID int64, transactionType string, delta int64, reference, note string, createdByID *int64, now time.Time) (*model.CreditTransaction, error) {
	var balance int64
	err := tx.QueryRow(ctx, `
		UPDATE companies
		SET credit_balance = credit_balance + $1
		WHERE id = $2 AND credit_balance + $1 >= 0
		RETURNING credit_balance
	`, delta, companyID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.Conflict("insufficient credit balance")
	}
	if err != nil {
		return nil, fmt.Errorf("update credit balance: %w", err)
	}
	transaction := &model.CreditTransaction{}
	err = tx.QueryRow(ctx, `
		INSERT INTO credit_transactions (company_id, type, delta_credits, balance_after, reference, note, created_by_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, company_id, type, delta_credits, balance_after, reference, note, created_by_id, created_at
	`, companyID, transactionType, delta, balance, reference, note, createdByID, now).Scan(
		&transaction.ID, &transaction.CompanyID, &transaction.Type, &transaction.DeltaCredits, &transaction.BalanceAfter,
		&transaction.Reference, &transaction.Note, &transaction.CreatedByID, &transaction.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("append credit transaction: %w", err)
	}
	return transaction, nil
}

func scanCompany(row rowScanner) (*model.Company, error) {
	company := &model.Company{}
	if err := row.Scan(&company.ID, &company.OwnerID, &company.Name, &company.CreditBalance, &company.CreatedAt); err != nil {
		return nil, err
	}
	return company, nil
}

func scanCreditPackage(row rowScanner) (*model.CreditPackage, error) {
	creditPackage := &model.CreditPackage{}
	if err := row.Scan(&creditPackage.ID, &creditPackage.Code, &creditPackage.Name, &creditPackage.PriceCents, &creditPackage.Currency, &creditPackage.Credits, &creditPackage.BonusCredits, &creditPackage.Active, &creditPackage.CreatedAt); err != nil {
		return nil, err
	}
	return creditPackage, nil
}

func scanPromotion(row rowScanner) (*model.Promotion, error) {
	promotion := &model.Promotion{}
	var code *string
	if err := row.Scan(&promotion.ID, &code, &promotion.Name, &promotion.Kind, &promotion.RewardType, &promotion.RewardValue, &promotion.StartsAt, &promotion.EndsAt, &promotion.MaxRedemptions, &promotion.TotalRedemptions, &promotion.Active, &promotion.CreatedAt); err != nil {
		return nil, err
	}
	if code != nil {
		promotion.Code = *code
	}
	return promotion, nil
}

func scanCreditTransaction(row rowScanner) (*model.CreditTransaction, error) {
	transaction := &model.CreditTransaction{}
	if err := row.Scan(&transaction.ID, &transaction.CompanyID, &transaction.Type, &transaction.DeltaCredits, &transaction.BalanceAfter, &transaction.Reference, &transaction.Note, &transaction.CreatedByID, &transaction.CreatedAt); err != nil {
		return nil, err
	}
	return transaction, nil
}

func scanCreditPurchase(row rowScanner) (*model.CreditPurchase, error) {
	purchase := &model.CreditPurchase{}
	if err := row.Scan(&purchase.ID, &purchase.CompanyID, &purchase.PackageID, &purchase.Status, &purchase.Provider, &purchase.ProviderReference, &purchase.AmountCents, &purchase.DiscountCents, &purchase.Currency, &purchase.Credits, &purchase.CreatedAt); err != nil {
		return nil, err
	}
	return purchase, nil
}

func normalizePromotionCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func mapPromotionWriteError(err error) error {
	if database.IsUniqueViolation(err) {
		return model.Conflict("promo code is already in use")
	}
	return fmt.Errorf("create promotion: %w", err)
}
