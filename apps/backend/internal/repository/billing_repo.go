package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BillingRepository owns all mutations of a company's credit balance. Balance
// updates and ledger inserts are intentionally performed in one transaction so
// reporting can always reconcile the cached balance to the immutable ledger.
type BillingRepository struct {
	pool *pgxpool.Pool
}

func NewBillingRepository(pool *pgxpool.Pool) *BillingRepository {
	return &BillingRepository{pool: pool}
}

func (r *BillingRepository) EnsureCompany(ctx context.Context, ownerID int64, name string) (*model.Company, error) {
	company, err := scanCompany(r.pool.QueryRow(ctx, `
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
	company, err := scanCompany(r.pool.QueryRow(ctx, `
		UPDATE companies
		SET name = $1
		WHERE owner_id = $2
		RETURNING id, owner_id, name, credit_balance, created_at
	`, name, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
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
	rows, err := r.pool.Query(ctx, `
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

func (r *BillingRepository) CompleteManualPurchase(ctx context.Context, ownerID, packageID int64, promoCode string, now time.Time) (*model.PurchaseCreditsResponse, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
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
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
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
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin credit adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedCompanyID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM companies WHERE id = $1 FOR UPDATE`, companyID).Scan(&lockedCompanyID); errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NotFound("company not found")
	} else if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}
	ledger, err := appendCredits(ctx, tx, companyID, model.CreditTransactionAdminAdjustment, delta, "admin:"+fmt.Sprint(adminID), note, &adminID, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credit adjustment: %w", err)
	}
	return ledger, nil
}

func (r *BillingRepository) companyByOwner(ctx context.Context, ownerID int64) (*model.Company, error) {
	company, err := scanCompany(r.pool.QueryRow(ctx, `
		SELECT id, owner_id, name, credit_balance, created_at FROM companies WHERE owner_id = $1
	`, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}
	return company, nil
}

func (r *BillingRepository) transactionsForCompany(ctx context.Context, companyID int64, limit int) ([]model.CreditTransaction, error) {
	rows, err := r.pool.Query(ctx, `
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
	rows, err := r.pool.Query(ctx, `
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

func companyByOwnerForUpdate(ctx context.Context, tx pgx.Tx, ownerID int64) (*model.Company, error) {
	company, err := scanCompany(tx.QueryRow(ctx, `
		SELECT id, owner_id, name, credit_balance, created_at FROM companies WHERE owner_id = $1 FOR UPDATE
	`, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NotFound("company not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}
	return company, nil
}

func activePackageForUpdate(ctx context.Context, tx pgx.Tx, packageID int64) (*model.CreditPackage, error) {
	creditPackage, err := scanCreditPackage(tx.QueryRow(ctx, `
		SELECT id, code, name, price_cents, currency, credits, bonus_credits, active, created_at
		FROM credit_packages WHERE id = $1 AND active = TRUE FOR UPDATE
	`, packageID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NotFound("credit package not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load credit package: %w", err)
	}
	return creditPackage, nil
}

func availablePromotionForUpdate(ctx context.Context, tx pgx.Tx, code string, now time.Time) (*model.Promotion, error) {
	promotion, err := scanPromotion(tx.QueryRow(ctx, `
		SELECT id, code, name, kind, reward_type, reward_value, starts_at, ends_at, max_redemptions, total_redemptions, active, created_at
		FROM promotions
		WHERE code = $1 AND active = TRUE
		  AND (starts_at IS NULL OR starts_at <= $2)
		  AND (ends_at IS NULL OR ends_at > $2)
		  AND (max_redemptions IS NULL OR total_redemptions < max_redemptions)
		FOR UPDATE
	`, code, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NotFound("promo code unavailable")
	}
	if err != nil {
		return nil, fmt.Errorf("load promotion: %w", err)
	}
	return promotion, nil
}

func assertNotRedeemed(ctx context.Context, tx pgx.Tx, promotionID, companyID int64) error {
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

func appendCredits(ctx context.Context, tx pgx.Tx, companyID int64, transactionType string, delta int64, reference, note string, createdByID *int64, now time.Time) (*model.CreditTransaction, error) {
	var balance int64
	err := tx.QueryRow(ctx, `
		UPDATE companies
		SET credit_balance = credit_balance + $1
		WHERE id = $2 AND credit_balance + $1 >= 0
		RETURNING credit_balance
	`, delta, companyID).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
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
