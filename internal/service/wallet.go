package service

import (
	"context"
	"fmt"
	"log/slog"
	"myApi/internal/storage/postgresql"
)

// Repository описывает методы основной БД (Postgres)
type Repository interface {
	CreateWallet(ctx context.Context, currency string) (string, error)
	GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error)
	Transfer(ctx context.Context, fromID, toID string, amount float64) error
}

// Cache описывает методы кэша (Redis)
type Cache interface {
	SetWallet(ctx context.Context, wallet *postgresql.Wallet) error
	GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error)
	DeleteWallet(ctx context.Context, walletID string) error
}

// WalletService объединяет базу и кэш
type WalletService struct {
	log   *slog.Logger
	repo  Repository
	cache Cache
}

func New(log *slog.Logger, repo Repository, cache Cache) *WalletService {
	return &WalletService{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// CreateWallet пока просто пробрасываем в БД (кэшировать пустой кошелек нет смысла)
func (s *WalletService) CreateWallet(ctx context.Context, currency string) (string, error) {
	return s.repo.CreateWallet(ctx, currency)
}

// GetWallet - А ВОТ ТУТ МАГИЯ КЭШИРОВАНИЯ
func (s *WalletService) GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error) {
	// ТВОЯ ЗАДАЧА: Написать логику здесь

	// 1. Пытаемся достать кошелек из кэша (s.cache.GetWallet)
	// Не забудь проверить ошибку. Если ошибка есть - просто залогируй её (s.log.Error),
	// но не прерывай работу (return не делай), ведь мы можем сходить в БД!
	wallet, err := s.cache.GetWallet(ctx, walletID)
	if err != nil {
		s.log.Error("failed to get wallet", slog.String("error", err.Error()))
	}
	// 2. Если кошелек нашелся в кэше (переменная кошелька != nil) ->
	// Залогируй "cache hit" (чтобы мы потом это увидели) и сразу верни кошелек и nil! БИНГО!
	if wallet != nil {
		s.log.Info("cache hit")
		return wallet, nil
	}
	// 3. Если мы дошли сюда, значит в кэше пусто. Идем в БД (s.repo.GetWallet).
	wallet, err = s.repo.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}
	// 4. Если БД вернула ошибку -> возвращаем эту ошибку (скорее всего это ErrWalletNotFound).
	if wallet != nil {
		err = s.cache.SetWallet(ctx, wallet)
		if err != nil {
			s.log.Error("error set wallet in cache", slog.String("error", err.Error()))
		}
	}
	// 5. Если БД успешно нашла кошелек -> сохраняем его в кэш для будущих запросов (s.cache.SetWallet).
	// Если при сохранении в кэш произошла ошибка - просто залогируй её, это не должно ломать ответ клиенту.
	s.log.Info("cache miss", slog.String("wallet_id", walletID))
	return wallet, nil
	// 6. Залогируй "cache miss" и верни найденный в БД кошелек.
}

func (s *WalletService) Transfer(ctx context.Context, fromID, toID string, amount float64) error {
	err := s.repo.Transfer(ctx, fromID, toID, amount)
	if err != nil {
		return fmt.Errorf("error to transfer : %w", err)
	}

	err = s.cache.DeleteWallet(ctx, fromID)
	if err != nil {
		s.log.Error("error delete wallet", slog.String("wallet_id", fromID))
	}

	err = s.cache.DeleteWallet(ctx, toID)
	if err != nil {
		s.log.Error("error delete wallet", slog.String("wallet_id", toID))
	}
	return nil
}
