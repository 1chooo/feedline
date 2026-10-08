package service

import (
	"context"

	"github.com/1chooo/ad-service/internal/model"
)

type AdminOperationsStore interface {
	ListUsers(ctx context.Context) ([]model.AdminUser, error)
	ListCompanies(ctx context.Context) ([]model.AdminCompany, error)
	ListCampaigns(ctx context.Context) ([]model.AdminCampaign, error)
	UpdateUserRole(ctx context.Context, id int64, role string) (*model.AdminUser, error)
	UpdateCampaignStatus(ctx context.Context, id int64, status string) (*model.AdminCampaign, error)
}

type AdminOperationsService struct {
	store AdminOperationsStore
}

func NewAdminOperationsService(store AdminOperationsStore) *AdminOperationsService {
	return &AdminOperationsService{store: store}
}

func (s *AdminOperationsService) Users(ctx context.Context) ([]model.AdminUser, error) {
	return s.store.ListUsers(ctx)
}

func (s *AdminOperationsService) Companies(ctx context.Context) ([]model.AdminCompany, error) {
	return s.store.ListCompanies(ctx)
}

func (s *AdminOperationsService) Campaigns(ctx context.Context) ([]model.AdminCampaign, error) {
	return s.store.ListCampaigns(ctx)
}

func (s *AdminOperationsService) SetUserRole(ctx context.Context, id int64, req model.UpdateUserRoleRequest) (*model.AdminUser, error) {
	if id < 1 {
		return nil, model.NotFound("user not found")
	}
	role, err := model.ValidateManagedRole(req.Role)
	if err != nil {
		return nil, err
	}
	return s.store.UpdateUserRole(ctx, id, role)
}

func (s *AdminOperationsService) SetCampaignStatus(ctx context.Context, id int64, req model.UpdateCampaignStatusRequest) (*model.AdminCampaign, error) {
	if id < 1 {
		return nil, model.NotFound("campaign not found")
	}
	status, err := model.ValidateManagedCampaignStatus(req.Status)
	if err != nil {
		return nil, err
	}
	return s.store.UpdateCampaignStatus(ctx, id, status)
}
