package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          uuid.UUID
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartDate   time.Time
	EndDate     *time.Time
}

type SubscriptionResponse struct {
	ID          uuid.UUID `json:"id"           example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	ServiceName string    `json:"service_name" example:"Yandex Plus"`
	Price       int       `json:"price"        example:"400"`
	UserID      uuid.UUID `json:"user_id"      example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string    `json:"start_date"   example:"07-2025"`
	EndDate     *string   `json:"end_date"     example:"12-2025"`
}

func ToResponse(s *Subscription) SubscriptionResponse {
	r := SubscriptionResponse{
		ID:          s.ID,
		ServiceName: s.ServiceName,
		Price:       s.Price,
		UserID:      s.UserID,
		StartDate:   FormatMonthYear(s.StartDate),
	}
	if s.EndDate != nil {
		str := FormatMonthYear(*s.EndDate)
		r.EndDate = &str
	}
	return r
}

type CreateRequest struct {
	ServiceName string    `json:"service_name" binding:"required,min=1,max=255" example:"Yandex Plus"`
	Price       int       `json:"price"        binding:"required,gt=0"          example:"400"`
	UserID      uuid.UUID `json:"user_id"      binding:"required"               example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string    `json:"start_date"   binding:"required"               example:"07-2025"`
	EndDate     *string   `json:"end_date"                                      example:"12-2025"`
}

// UpdateRequest — все поля опциональны.
// EndDate: отсутствие = не менять; null = убрать дату; "MM-YYYY" = новое значение.
type UpdateRequest struct {
	ServiceName *string         `json:"service_name"                    example:"Netflix"`
	Price       *int            `json:"price"                           example:"599"`
	EndDate     json.RawMessage `json:"end_date" swaggertype:"string"   example:"12-2025"`
}

type TotalCostResponse struct {
	TotalCost          int64      `json:"total_cost"           example:"4800"`
	PeriodStart        string     `json:"period_start"         example:"01-2025"`
	PeriodEnd          string     `json:"period_end"           example:"12-2025"`
	SubscriptionsCount int        `json:"subscriptions_count"  example:"2"`
	UserID             *uuid.UUID `json:"user_id,omitempty"    example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	ServiceName        *string    `json:"service_name,omitempty" example:"Yandex Plus"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"подписка не найдена"`
}

func ParseMonthYear(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("неверный формат даты %q, ожидается MM-YYYY", s)
	}
	return t, nil
}

func FormatMonthYear(t time.Time) string {
	return t.Format("01-2006")
}
