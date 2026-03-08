package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	// Импортируй свою структуру Wallet (из пакета postgresql)
	"myApi/internal/storage/postgresql"
)

type Cache struct {
	client *redis.Client
}

// New создает подключение к Redis
func New(addr string) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Проверяем подключение (Ping)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// SetWallet сохраняет кошелек в кэш
func (c *Cache) SetWallet(ctx context.Context, wallet *postgresql.Wallet) error {
	// 1. Так как Redis хранит только строки/байты, нам нужно превратить структуру Wallet в JSON.
	// Используй json.Marshal(wallet)
	data, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet: %w", err)
	}
	// 2. Сохрани полученные байты в Redis. Ключом будет ID кошелька.
	// Используй c.client.Set(ctx, "wallet:"+wallet.ID, jsonBytes, 5*time.Minute).Err()
	// (Обрати внимание, мы ставим срок жизни кэша TTL = 5 минут. Это защита от "вечно старых" данных).
	key := "wallet:" + wallet.ID
	err = c.client.Set(ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil

	// 3. Обработай ошибку и верни её
}

// GetWallet достает кошелек из кэша
func (c *Cache) GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error) {
	// 1. Прочитай данные по ключу "wallet:"+walletID
	// val, err := c.client.Get(ctx, "wallet:"+walletID).Bytes()
	key := "wallet:" + walletID
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get cache: %w", err)
	}
	// 2. Если ошибка == redis.Nil - значит в кэше пусто. Возвращаем (nil, nil) - это не ошибка, просто данных нет.
	// Если другая ошибка - возвращаем её.
	var w postgresql.Wallet
	if err := json.Unmarshal(val, &w); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet: %w", err)
	}
	return &w, nil
	// 3. Распарси байты обратно в структуру Wallet через json.Unmarshal
	// Верни указатель на Wallet
}

// DeleteWallet удаляет кошелек из кэша (Инвалидация)
func (c *Cache) DeleteWallet(ctx context.Context, walletID string) error {
	key := "wallet:" + walletID

	// 1. Используй c.client.Del(ctx, "wallet:"+walletID).Err()
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
	// 2. Верни ошибку, если она есть
}
