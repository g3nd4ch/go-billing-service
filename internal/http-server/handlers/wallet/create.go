package wallet

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// Request описывает входящий JSON
type Request struct {
	Currency string `json:"currency"`
}

// Response описывает исходящий JSON
type Response struct {
	WalletID string `json:"wallet_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// WalletCreator - интерфейс. Хендлеру всё равно, кто это делает (Postgres, мок для тестов или файл),
// главное, чтобы у объекта был метод CreateWallet.
type WalletCreator interface {
	CreateWallet(ctx context.Context, currency string) (string, error)
}

// New создает HTTP-хендлер для создания кошелька
func New(log *slog.Logger, creator WalletCreator) http.HandlerFunc {
	// Возвращаем саму функцию-обработчик (замыкание)
	return func(w http.ResponseWriter, r *http.Request) {
		// ТВОЯ ЗАДАЧА: Написать логику здесь!
		var req Request
		// 1. Создать пустую переменную req типа Request

		// 2. Распарсить входящий JSON.
		// Подсказка: err := json.NewDecoder(r.Body).Decode(&req)
		// Если ошибка - логируем, пишем статус 400 (w.WriteHeader(http.StatusBadRequest)) и выходим (return)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", slog.String("error", err.Error()))
			http.Error(w, "error decode json", http.StatusBadRequest)
			return
		}

		// 3. Вызвать бизнес-логику (создание кошелька).
		// Подсказка: walletID, err := creator.CreateWallet(r.Context(), req.Currency)
		// Если ошибка - логируем, ставим статус 500, пишем JSON с ошибкой и выходим.
		walletID, err := creator.CreateWallet(r.Context(), req.Currency)
		if err != nil {
			log.Error("failed to create wallet", slog.String("error", err.Error()))
			http.Error(w, "error create wallet", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := Response{
			WalletID: walletID,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil{
			http.Error(w, "error encode json", http.StatusInternalServerError)
			return
		}
		// 4. Если всё успешно:
		// Установить заголовок ответа: w.Header().Set("Content-Type", "application/json")
		// Установить статус 201 Created: w.WriteHeader(http.StatusCreated)
		// Сформировать структуру Response и отправить её клиенту: json.NewEncoder(w).Encode(response)
	}
}
