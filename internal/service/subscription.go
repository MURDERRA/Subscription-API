package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"subscription-service/internal/model"
	"subscription-service/internal/repository"
)

var ErrNotFound = repository.ErrNotFound

type Service struct {
	repo *repository.Repo
}

func New(repo *repository.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req *model.CreateRequest) (*model.Subscription, error) {
	startDate, err := model.ParseMonthYear(req.StartDate)
	if err != nil {
		return nil, err
	}

	sub := &model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   startDate,
	}

	if req.EndDate != nil {
		endDate, err := model.ParseMonthYear(*req.EndDate)
		if err != nil {
			return nil, err
		}
		if endDate.Before(startDate) {
			return nil, errors.New("end_date не может быть раньше start_date")
		}
		sub.EndDate = &endDate
	}

	return s.repo.Create(ctx, sub)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, userID *uuid.UUID, serviceName *string, limit, offset int) ([]*model.Subscription, error) {
	return s.repo.List(ctx, repository.ListFilter{
		UserID:      userID,
		ServiceName: serviceName,
		Limit:       limit,
		Offset:      offset,
	})
}

type UpdateInput struct {
	ID          uuid.UUID
	ServiceName *string
	Price       *int
	EndDateSet  bool
	EndDate     *time.Time
}

func (s *Service) Update(ctx context.Context, in UpdateInput) (*model.Subscription, error) {
	if in.EndDateSet && in.EndDate != nil {
		current, err := s.repo.GetByID(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if in.EndDate.Before(current.StartDate) {
			return nil, fmt.Errorf("end_date не может быть раньше start_date (%s)", model.FormatMonthYear(current.StartDate))
		}
	}

	return s.repo.Update(ctx, repository.UpdateParams{
		ID:          in.ID,
		ServiceName: in.ServiceName,
		Price:       in.Price,
		EndDateSet:  in.EndDateSet,
		EndDate:     in.EndDate,
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

type TotalCostInput struct {
	PeriodStart string
	PeriodEnd   string
	UserID      *uuid.UUID
	ServiceName *string
}

func (s *Service) TotalCost(ctx context.Context, in TotalCostInput) (*model.TotalCostResponse, error) {
	ps, err := model.ParseMonthYear(in.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("period_start: %w", err)
	}
	pe, err := model.ParseMonthYear(in.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("period_end: %w", err)
	}
	if pe.Before(ps) {
		return nil, errors.New("period_end не может быть раньше period_start")
	}

	total, count, err := s.repo.TotalCost(ctx, repository.TotalCostParams{
		PeriodStart: ps,
		PeriodEnd:   pe,
		UserID:      in.UserID,
		ServiceName: in.ServiceName,
	})
	if err != nil {
		return nil, err
	}

	return &model.TotalCostResponse{
		TotalCost:          total,
		PeriodStart:        in.PeriodStart,
		PeriodEnd:          in.PeriodEnd,
		SubscriptionsCount: count,
		UserID:             in.UserID,
		ServiceName:        in.ServiceName,
	}, nil
}
