package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"subscription-service/internal/model"
)

var ErrNotFound = errors.New("не найдено")

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

type UpdateParams struct {
	ID          uuid.UUID
	ServiceName *string
	Price       *int
	EndDateSet  bool // true — значит поле end_date присутствовало в теле запроса
	EndDate     *time.Time
}

type TotalCostParams struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	UserID      *uuid.UUID
	ServiceName *string
}

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// scanRow сканирует одну строку subscriptions.
func scanRow(row *sql.Row) (*model.Subscription, error) {
	var (
		idStr, userIDStr string
		s                model.Subscription
		endDate          *time.Time
	)
	err := row.Scan(&idStr, &s.ServiceName, &s.Price, &userIDStr, &s.StartDate, &endDate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.ID, _ = uuid.Parse(idStr)
	s.UserID, _ = uuid.Parse(userIDStr)
	s.EndDate = endDate
	return &s, nil
}

func (r *Repo) Create(ctx context.Context, s *model.Subscription) (*model.Subscription, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price, user_id, start_date, end_date`,
		s.ServiceName, s.Price, s.UserID.String(), s.StartDate, s.EndDate,
	)
	return scanRow(row)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions WHERE id = $1`, id.String())
	return scanRow(row)
}

func (r *Repo) List(ctx context.Context, f ListFilter) ([]*model.Subscription, error) {
	args := []any{}
	conds := []string{}

	if f.UserID != nil {
		args = append(args, f.UserID.String())
		conds = append(conds, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if f.ServiceName != nil {
		args = append(args, "%"+*f.ServiceName+"%")
		conds = append(conds, fmt.Sprintf("service_name ILIKE $%d", len(args)))
	}

	where := "1=1"
	if len(conds) > 0 {
		where = strings.Join(conds, " AND ")
	}

	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf(`
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions
		WHERE %s
		ORDER BY start_date DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Subscription
	for rows.Next() {
		var (
			idStr, userIDStr string
			s                model.Subscription
			endDate          *time.Time
		)
		if err := rows.Scan(&idStr, &s.ServiceName, &s.Price, &userIDStr, &s.StartDate, &endDate); err != nil {
			return nil, err
		}
		s.ID, _ = uuid.Parse(idStr)
		s.UserID, _ = uuid.Parse(userIDStr)
		s.EndDate = endDate
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (r *Repo) Update(ctx context.Context, p UpdateParams) (*model.Subscription, error) {
	sets := []string{}
	args := []any{}

	if p.ServiceName != nil {
		args = append(args, *p.ServiceName)
		sets = append(sets, fmt.Sprintf("service_name = $%d", len(args)))
	}
	if p.Price != nil {
		args = append(args, *p.Price)
		sets = append(sets, fmt.Sprintf("price = $%d", len(args)))
	}
	if p.EndDateSet {
		if p.EndDate == nil {
			sets = append(sets, "end_date = NULL")
		} else {
			args = append(args, *p.EndDate)
			sets = append(sets, fmt.Sprintf("end_date = $%d", len(args)))
		}
	}

	// Если нечего обновлять — возвращаемс текущую запись
	if len(sets) == 0 {
		return r.GetByID(ctx, p.ID)
	}

	args = append(args, p.ID.String())
	q := fmt.Sprintf(`
		UPDATE subscriptions SET %s WHERE id = $%d
		RETURNING id, service_name, price, user_id, start_date, end_date`,
		strings.Join(sets, ", "), len(args))

	return scanRow(r.db.QueryRowContext(ctx, q, args...))
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = $1`, id.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) TotalCost(ctx context.Context, p TotalCostParams) (totalCost int64, count int, err error) {
	args := []any{p.PeriodStart, p.PeriodEnd}
	where := "start_date <= $2 AND (end_date IS NULL OR end_date >= $1)"

	if p.UserID != nil {
		args = append(args, p.UserID.String())
		where += fmt.Sprintf(" AND user_id = $%d", len(args))
	}
	if p.ServiceName != nil {
		args = append(args, "%"+*p.ServiceName+"%")
		where += fmt.Sprintf(" AND service_name ILIKE $%d", len(args))
	}

	// AGE(later, earlier) возвращает interval, из него берём полные года и месяцы
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(
				price * (
					(EXTRACT(YEAR FROM AGE(
						LEAST(COALESCE(end_date, $2::date), $2::date),
						GREATEST(start_date, $1::date)
					)) * 12 +
					EXTRACT(MONTH FROM AGE(
						LEAST(COALESCE(end_date, $2::date), $2::date),
						GREATEST(start_date, $1::date)
					)) + 1)::bigint
				)
			), 0)::bigint,
			COUNT(*)::int
		FROM subscriptions
		WHERE %s`, where)

	err = r.db.QueryRowContext(ctx, q, args...).Scan(&totalCost, &count)
	return
}
