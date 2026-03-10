package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"myApi/internal/storage/postgresql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- 1. СОЗДАЕМ ФЕЙКОВЫЙ КЭШ ---
type MockCache struct {
	mock.Mock
}

func (m *MockCache) GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error) {
	// Эта магия позволяет нам в тесте сказать: "Когда вызовут этот метод, верни вот эти данные"
	args := m.Called(ctx, walletID)

	// Если мы сказали вернуть nil (данных нет)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// Если данные есть, приводим их к типу *Wallet
	return args.Get(0).(*postgresql.Wallet), args.Error(1)
}

func (m *MockCache) SetWallet(ctx context.Context, wallet *postgresql.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockCache) DeleteWallet(ctx context.Context, walletID string) error {
	args := m.Called(ctx, walletID)
	return args.Error(0)
}

// --- 2. СОЗДАЕМ ФЕЙКОВУЮ БАЗУ (Только нужные методы) ---
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error) {
	args := m.Called(ctx, walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*postgresql.Wallet), args.Error(1)
}
func (m *MockRepo) CreateWallet(ctx context.Context, currency string) (string, error) { return "", nil }
func (m *MockRepo) Transfer(ctx context.Context, fromID, toID string, amount float64) error {
	// Эта строчка записывает в журнал testify, что метод был вызван с такими-то аргументами!
	args := m.Called(ctx, fromID, toID, amount)

	// Возвращаем ту ошибку, которую мы указали в тесте через .Return()
	return args.Error(0)
}

// --- 3. САМ ТЕСТ ---
func TestWalletService_GetWallet_CacheHit(t *testing.T) {
	// Подготовка (Arrange)
	mockRepo := new(MockRepo)
	mockCache := new(MockCache)
	// (Для логгера передаем nil или создаем пустой, чтобы не спамил в тесте)
	dummyLog := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := New(dummyLog, mockRepo, mockCache)

	testWalletID := "123-uuid"
	expectedWallet := &postgresql.Wallet{
		ID:       testWalletID,
		Balance:  500,
		Currency: "RUB",
	}

	// ТВОЯ ЗАДАЧА: Настроить поведение моков и вызвать сервис

	// 1. Скажи фейковому кэшу, чтобы он при вызове GetWallet вернул expectedWallet и nil в качестве ошибки.
	// Синтаксис testify:
	// mockCache.On("GetWallet", mock.Anything, testWalletID).Return(expectedWallet, nil)
	mockCache.On("GetWallet", mock.Anything, testWalletID).Return(expectedWallet, nil)
	// 2. Скажи фейковой базе данных (mockRepo), что метод GetWallet вообще НЕ ДОЛЖЕН вызываться!
	// (ведь мы ожидаем Cache Hit).
	// Если код пойдет в базу, тест должен упасть с ошибкой.
	// Подсказка: mockRepo.AssertNotCalled(t, "GetWallet", mock.Anything, testWalletID)
	// Эту проверку (AssertNotCalled) нужно делать В САМОМ КОНЦЕ теста.

	// 3. Выполни действие (Act): вызови service.GetWallet(context.Background(), testWalletID)
	// Сохрани результаты в переменные resultWallet, err
	resultWallet, err := service.GetWallet(context.Background(), testWalletID)
	// 4. Проверь результат (Assert):
	// - Ошибка должна быть nil: assert.NoError(t, err)
	// - Вернувшийся кошелек должен совпадать с ожидаемым: assert.Equal(t, expectedWallet, resultWallet)
	assert.NoError(t, err)
	assert.Equal(t, expectedWallet, resultWallet)
	mockRepo.AssertNotCalled(t, "GetWallet", mock.Anything, testWalletID)
}

func TestWalletService_Transfer_Success(t *testing.T) {
	// Подготовка (Arrange)
	dummyLog := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockRepo := new(MockRepo)
	mockCache := new(MockCache)
	service := New(dummyLog, mockRepo, mockCache)

	fromID := "wallet-1"
	toID := "wallet-2"
	amount := float64(100)

	// ТВОЯ ЗАДАЧА: Срежиссировать сценарий для Моков

	// 1. Скажи БД (mockRepo): когда вызовут твой метод "Transfer" с нашими fromID, toID и amount, верни nil (типа перевод успешен).
	// Подсказка: mockRepo.On("Transfer", mock.Anything, fromID, toID, amount).Return(nil)
	mockRepo.On("Transfer", mock.Anything, fromID, toID, amount).Return(nil)
	// 2. Скажи Кэшу (mockCache): я ожидаю, что тебя попросят удалить кошелек fromID. Верни nil.
	// Подсказка: mockCache.On("DeleteWallet", mock.Anything, fromID).Return(nil)
	mockCache.On("DeleteWallet", mock.Anything, fromID).Return(nil)
	// 3. Скажи Кэшу (mockCache): я ожидаю, что тебя попросят удалить кошелек toID. Верни nil.
	mockCache.On("DeleteWallet", mock.Anything, toID).Return(nil)

	// 4. Вызови тестируемый метод (Act): err := service.Transfer(...)

	err := service.Transfer(context.Background(), fromID, toID, amount)
	// 5. Проверь результат (Assert): убедись, что ошибка равна nil (assert.NoError)
	assert.NoError(t, err)
	// 6. ФИНАЛЬНАЯ ПРОВЕРКА (Ревизор)
	// Эти две строчки проверяют, что сервис реально выполнил все условия из пунктов 1, 2 и 3!
	// Если сервис "забыл" удалить чей-то кэш, тест упадет с криком "Вы обещали вызвать DeleteWallet, но не вызвали!".
	// Напиши:
	// mockRepo.AssertExpectations(t)
	// mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}
