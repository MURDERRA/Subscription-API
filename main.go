package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "subscription-service/docs" // swag init -g main.go -o docs

	"subscription-service/internal/config"
	"subscription-service/internal/db"
	"subscription-service/internal/handler"
	"subscription-service/internal/logger"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"
)

// @title           Subscription Service
// @version         1.0
// @description     API управления подписками пользователей
// @host            localhost:8000
// @BasePath        /

func main() {
	cfg := config.Load()

	log := logger.New(cfg.LogLevel, cfg.LogDir)
	slog.SetDefault(log)

	slog.Info("старт сервиса", "port", cfg.Port)

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("ошибка подключения к БД", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("ошибка миграций", "err", err)
		os.Exit(1)
	}

	repo := repository.New(database)
	svc := service.New(repo)
	h := handler.New(svc, log)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.SetupRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("сервер запущен", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("ошибка сервера", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("завершение работы...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("ошибка при остановке", "err", err)
	}
	slog.Info("сервер остановлен")
}
