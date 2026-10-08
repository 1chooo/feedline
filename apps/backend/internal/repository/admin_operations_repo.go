package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

type AdminOperationsRepository struct {
	db database.DB
}

func NewAdminOperationsRepository(db database.DB) *AdminOperationsRepository {
	return &AdminOperationsRepository{db: db}
}

func (r *AdminOperationsRepository) ListUsers(ctx context.Context) ([]model.AdminUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, email, display_name, role, created_at
		FROM users ORDER BY created_at DESC, id DESC LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()
	users := []model.AdminUser{}
	for rows.Next() {
		var user model.AdminUser
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan admin user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin users: %w", err)
	}
	return users, nil
}

func (r *AdminOperationsRepository) ListCompanies(ctx context.Context) ([]model.AdminCompany, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name, c.credit_balance, u.username, u.email, c.created_at
		FROM companies c JOIN users u ON u.id = c.owner_id
		ORDER BY c.created_at DESC, c.id DESC LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("list admin companies: %w", err)
	}
	defer rows.Close()
	companies := []model.AdminCompany{}
	for rows.Next() {
		var company model.AdminCompany
		if err := rows.Scan(&company.ID, &company.Name, &company.CreditBalance, &company.OwnerUsername, &company.OwnerEmail, &company.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan admin company: %w", err)
		}
		companies = append(companies, company)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin companies: %w", err)
	}
	return companies, nil
}

func (r *AdminOperationsRepository) ListCampaigns(ctx context.Context) ([]model.AdminCampaign, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ad.id, ad.title, ad.status, COALESCE(u.username, ''), COALESCE(c.name, ''), ad.credit_budget, ad.credit_spent, ad.start_at, ad.end_at, ad.created_at
		FROM ads ad
		LEFT JOIN users u ON u.id = ad.advertiser_id
		LEFT JOIN companies c ON c.id = ad.company_id
		ORDER BY ad.created_at DESC, ad.id DESC LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("list admin campaigns: %w", err)
	}
	defer rows.Close()
	campaigns := []model.AdminCampaign{}
	for rows.Next() {
		var campaign model.AdminCampaign
		if err := rows.Scan(&campaign.ID, &campaign.Title, &campaign.Status, &campaign.AdvertiserUsername, &campaign.CompanyName, &campaign.CreditBudget, &campaign.CreditSpent, &campaign.StartAt, &campaign.EndAt, &campaign.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan admin campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin campaigns: %w", err)
	}
	return campaigns, nil
}

func (r *AdminOperationsRepository) UpdateUserRole(ctx context.Context, id int64, role string) (*model.AdminUser, error) {
	user := &model.AdminUser{}
	err := r.db.QueryRow(ctx, `
		UPDATE users SET role = $1 WHERE id = $2
		RETURNING id, username, email, display_name, role, created_at
	`, role, id).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update user role: %w", err)
	}
	return user, nil
}

func (r *AdminOperationsRepository) UpdateCampaignStatus(ctx context.Context, id int64, status string) (*model.AdminCampaign, error) {
	campaign := &model.AdminCampaign{}
	err := r.db.QueryRow(ctx, `
		UPDATE ads SET status = $1 WHERE id = $2
		RETURNING id, title, status,
			COALESCE((SELECT username FROM users WHERE users.id = ads.advertiser_id), ''),
			COALESCE((SELECT name FROM companies WHERE companies.id = ads.company_id), ''),
			credit_budget, credit_spent, start_at, end_at, created_at
	`, status, id).Scan(&campaign.ID, &campaign.Title, &campaign.Status, &campaign.AdvertiserUsername, &campaign.CompanyName, &campaign.CreditBudget, &campaign.CreditSpent, &campaign.StartAt, &campaign.EndAt, &campaign.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NotFound("campaign not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update campaign status: %w", err)
	}
	return campaign, nil
}
