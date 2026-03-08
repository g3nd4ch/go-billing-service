package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Storage struct {
	conn *pgxpool.Pool
}

func New(storagePath string) (*Storage, error) {
	pool, err := pgxpool.New(context.Background(), storagePath)
	if err != nil {
		return nil, fmt.Errorf("текст ошибки: %v", err)

	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping db: %w", err)

	}

	return &Storage{
		conn: pool,
	}, nil

}

func (s *Storage) Close() {
	s.conn.Close()
}

// создание кошелька
func (s *Storage) CreateWallet(ctx context.Context, currency string) (string, error) {
	id := uuid.New().String()
	query := "INSERT INTO wallets (id, currency) VALUES ($1, $2)"
	_, err := s.conn.Exec(ctx, query, id, currency)
	if err != nil {
		return "", fmt.Errorf("err to insert wallet: %w", err)
	}
	return id, nil
}

// получение кошелька
func (s *Storage) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	query := "SELECT id, balance, currency, created_at, updated_at FROM wallets WHERE id = $1"
	var w Wallet
	err := s.conn.QueryRow(ctx, query, walletID).Scan(&w.ID, &w.Balance, &w.Currency, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet:%w", err)
	}
	return &w, nil
}

// тразакция
func (s *Storage) Transfer(ctx context.Context, fromID, toID string, amount float64) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error transaction begin:%w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var balance float64
	query := "SELECT balance FROM wallets WHERE id = $1 FOR UPDATE"
	err = tx.QueryRow(ctx, query, fromID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("error scan balance:%w", err)
	}
	if balance < amount {
		return ErrInsufficientFunds
	}
	_, err = tx.Exec(ctx, "UPDATE wallets SET balance=balance - $2 WHERE id = $1", fromID, amount)
	if err != nil {
		return fmt.Errorf("error update row fromID: %w", err)
	}
	_, err = tx.Exec(ctx, "UPDATE wallets SET balance=balance + $2 WHERE id = $1", toID, amount)
	if err != nil {
		return fmt.Errorf("error update row toID:%w", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("error transaction:%w", err)
	}
	return nil
}

// пооплнение счета
func (s *Storage) Deposit(ctx context.Context, walletID string, amount float64) error {
	_, err := s.conn.Exec(ctx, "UPDATE wallets SET balance= balance + $1 WHERE id = $2", amount, walletID)
	if err != nil {
		return fmt.Errorf("error update balance:%w", err)
	}
	return nil
}
