package main

import (
	"context"
	"errors"
	"log/slog"
	"myApi/internal/config" // Поправь путь на свой модуль!
	"myApi/internal/http-server/handlers/wallet"
	"myApi/internal/storage/postgresql"
	"myApi/internal/storage/redis"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	// 1. Инициализация конфига
	cfg := config.MustLoad()

	// 2. Инициализация логгера
	log := setupLogger(cfg.Env)

	log.Info("starting billing service", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	// 3. Подключение к БД (В идеале dsn должен браться из cfg.StoragePath)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:mysecretpassword@localhost:5432/billing?sslmode=disable"
	}

	storage, err := postgresql.New(dsn)
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer storage.Close() // Закроет соединение перед выходом из main

	log.Info("successfully connected to database")

	cache, err := redis.New(cfg.RedisAddr)
	if err != nil {
		log.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	_ = cache
	log.Info("successfully connected to redis")
	// TODO: Здесь будет запуск HTTP-сервера
	router := chi.NewRouter()

	// 2. Добавляем полезные Middleware (промежуточные слои)
	router.Use(middleware.RequestID) // Добавляет ID к каждому запросу
	router.Use(middleware.Logger)    // Логирует все входящие HTTP запросы
	router.Use(middleware.Recoverer) // Спасает сервер от паники (чтобы не упал весь сервис)

	// 3. Регистрируем наши ручки (Endpoints)
	// Вот она - магия интерфейсов! Мы передаем storage, и он работает как WalletCreator.
	// Замени "myApi/internal/http-server/handlers/wallet" на свой путь, если IDE не импортирует сама!
	router.Post("/api/v1/wallet", wallet.New(log, storage))
	// {id} - это переменная пути. Роутер chi поймет, что вместо неё будет подставлен UUID
	router.Get("/api/v1/wallet/{id}", wallet.Get(log, storage))
	router.Post("/api/v1/wallet/transfer", wallet.Transfer(log, storage))

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}
	// Создаем канал, который ждет сигнал от операционной системы
	stop := make(chan os.Signal, 1)

	// Говорим пакету signal отправлять уведомления в канал stop
	// Ловим os.Interrupt (Ctrl+C) и syscall.SIGTERM (сигнал от Docker/Kubernetes)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	// Запускаем сервер в горутине
	go func() {
		// Ошибка http.ErrServerClosed - это норма, она возникает при вызове Shutdown
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	log.Info("server started", slog.String("address", cfg.Address))

	// Программа дойдет до этой строчки и "зависнет", ожидая сообщение из канала.
	// Как только ты нажмешь Ctrl+C, в канал прилетит сигнал, и код пойдет дальше.
	<-stop

	log.Info("stopping server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Приказываем серверу завершить работу
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", slog.String("error", err.Error()))

	}
	// Когда Shutdown завершится, сработает defer storage.Close(), который мы писали в самом начале main()!
	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		// Для локальной разработки - текстовый формат, уровень Debug
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		// Для Dev стенда - JSON, уровень Debug
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		// Для прода - JSON, только Info и выше (экономим место)
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
