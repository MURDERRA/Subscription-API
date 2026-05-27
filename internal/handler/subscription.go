package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"

	"subscription-service/internal/model"
	"subscription-service/internal/service"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func New(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) SetupRoutes() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(h.requestLogger(), gin.Recovery())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1/subscriptions")
	{
		v1.POST("", h.Create)
		v1.GET("", h.List)
		v1.GET("/total-cost", h.TotalCost) // до /:id
		v1.GET("/:id", h.GetByID)
		v1.PUT("/:id", h.Update)
		v1.DELETE("/:id", h.Delete)
	}

	return r
}

func (h *Handler) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		h.log.Info("http",
			"method", c.Request.Method,
			"path",   c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip",     c.ClientIP(),
		)
	}
}

func (h *Handler) fail(c *gin.Context, status int, msg string) {
	c.JSON(status, model.ErrorResponse{Error: msg})
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "неверный формат UUID"})
		return uuid.UUID{}, false
	}
	return id, true
}

// Create godoc
// @Summary      Создать подписку
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        body  body      model.CreateRequest        true  "Данные подписки"
// @Success      201   {object}  model.SubscriptionResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Router       /api/v1/subscriptions [post]
func (h *Handler) Create(c *gin.Context) {
	var req model.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, http.StatusBadRequest, err.Error())
		return
	}

	h.log.Info("создание подписки", "service", req.ServiceName, "user_id", req.UserID)

	sub, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		h.fail(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.log.Info("подписка создана", "id", sub.ID)
	c.JSON(http.StatusCreated, model.ToResponse(sub))
}

// List godoc
// @Summary      Список подписок
// @Tags         subscriptions
// @Produce      json
// @Param        user_id       query     string  false  "UUID пользователя"
// @Param        service_name  query     string  false  "Название сервиса (подстрока)"
// @Param        limit         query     int     false  "Лимит (макс 1000)"    default(100)
// @Param        offset        query     int     false  "Смещение"             default(0)
// @Success      200   {array}   model.SubscriptionResponse
// @Router       /api/v1/subscriptions [get]
func (h *Handler) List(c *gin.Context) {
	var userID *uuid.UUID
	if raw := c.Query("user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			h.fail(c, http.StatusBadRequest, "неверный формат user_id")
			return
		}
		userID = &id
	}

	var serviceName *string
	if raw := c.Query("service_name"); raw != "" {
		serviceName = &raw
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	h.log.Info("список подписок", "user_id", userID, "service_name", serviceName)

	subs, err := h.svc.List(c.Request.Context(), userID, serviceName, limit, offset)
	if err != nil {
		h.fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]model.SubscriptionResponse, 0, len(subs))
	for _, s := range subs {
		resp = append(resp, model.ToResponse(s))
	}
	c.JSON(http.StatusOK, resp)
}

// TotalCost godoc
// @Summary      Суммарная стоимость подписок за период
// @Description  Для каждой подходящей подписки: price × кол-во месяцев в периоде
// @Tags         analytics
// @Produce      json
// @Param        period_start  query     string  true   "Начало периода MM-YYYY"  example(01-2025)
// @Param        period_end    query     string  true   "Конец периода MM-YYYY"   example(12-2025)
// @Param        user_id       query     string  false  "UUID пользователя"
// @Param        service_name  query     string  false  "Название сервиса"
// @Success      200   {object}  model.TotalCostResponse
// @Failure      400   {object}  model.ErrorResponse
// @Router       /api/v1/subscriptions/total-cost [get]
func (h *Handler) TotalCost(c *gin.Context) {
	ps := c.Query("period_start")
	pe := c.Query("period_end")
	if ps == "" || pe == "" {
		h.fail(c, http.StatusBadRequest, "period_start и period_end обязательны")
		return
	}

	var userID *uuid.UUID
	if raw := c.Query("user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			h.fail(c, http.StatusBadRequest, "неверный формат user_id")
			return
		}
		userID = &id
	}

	var serviceName *string
	if raw := c.Query("service_name"); raw != "" {
		serviceName = &raw
	}

	h.log.Info("подсчёт стоимости", "period_start", ps, "period_end", pe)

	resp, err := h.svc.TotalCost(c.Request.Context(), service.TotalCostInput{
		PeriodStart: ps,
		PeriodEnd:   pe,
		UserID:      userID,
		ServiceName: serviceName,
	})
	if err != nil {
		h.fail(c, http.StatusBadRequest, err.Error())
		return
	}

	h.log.Info("стоимость подсчитана", "total", resp.TotalCost, "count", resp.SubscriptionsCount)
	c.JSON(http.StatusOK, resp)
}

// GetByID godoc
// @Summary      Получить подписку по ID
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "UUID подписки"
// @Success      200  {object}  model.SubscriptionResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /api/v1/subscriptions/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	h.log.Info("получение подписки", "id", id)

	sub, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.fail(c, http.StatusNotFound, "подписка не найдена")
			return
		}
		h.fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, model.ToResponse(sub))
}

// Update godoc
// @Summary      Обновить подписку
// @Description  Все поля опциональны. end_date: null — убрать дату; "MM-YYYY" — установить новую.
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "UUID подписки"
// @Param        body  body      model.UpdateRequest   true  "Данные для обновления"
// @Success      200   {object}  model.SubscriptionResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Router       /api/v1/subscriptions/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var body model.UpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if body.Price != nil && *body.Price <= 0 {
		h.fail(c, http.StatusBadRequest, "price должен быть больше 0")
		return
	}
	if body.ServiceName != nil && (len(*body.ServiceName) == 0 || len(*body.ServiceName) > 255) {
		h.fail(c, http.StatusBadRequest, "service_name: от 1 до 255 символов")
		return
	}

	in := service.UpdateInput{ID: id, ServiceName: body.ServiceName, Price: body.Price}

	if body.EndDate != nil {
		in.EndDateSet = true
		if string(body.EndDate) != "null" {
			var dateStr string
			if err := json.Unmarshal(body.EndDate, &dateStr); err != nil {
				h.fail(c, http.StatusBadRequest, "неверный формат end_date")
				return
			}
			t, err := model.ParseMonthYear(dateStr)
			if err != nil {
				h.fail(c, http.StatusBadRequest, err.Error())
				return
			}
			in.EndDate = &t
		}
	}

	h.log.Info("обновление подписки", "id", id)

	sub, err := h.svc.Update(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.fail(c, http.StatusNotFound, "подписка не найдена")
			return
		}
		h.fail(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.log.Info("подписка обновлена", "id", id)
	c.JSON(http.StatusOK, model.ToResponse(sub))
}

// Delete godoc
// @Summary      Удалить подписку
// @Tags         subscriptions
// @Param        id   path  string  true  "UUID подписки"
// @Success      204
// @Failure      404  {object}  model.ErrorResponse
// @Router       /api/v1/subscriptions/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	h.log.Info("удаление подписки", "id", id)

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.fail(c, http.StatusNotFound, "подписка не найдена")
			return
		}
		h.fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.log.Info("подписка удалена", "id", id)
	c.Status(http.StatusNoContent)
}
