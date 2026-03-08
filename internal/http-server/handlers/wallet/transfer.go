package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"myApi/internal/storage/postgresql"
	"net/http"
)

// Request описывает входящий JSON для перевода
type TransferRequest struct {
	FromWalletID string  `json:"from_wallet_id"`
	ToWalletID   string  `json:"to_wallet_id"`
	Amount       float64 `json:"amount"`
}

// WalletTransferrer - интерфейс для перевода
type WalletTransferrer interface {
	Transfer(ctx context.Context, fromID, toID string, amount float64) error
}

// Transfer создает HTTP-хендлер для перевода денег
func Transfer(log *slog.Logger, transferrer WalletTransferrer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ТВОЯ ЗАДАЧА:
		// 1. Создать переменную req типа TransferRequest

		var req TransferRequest
		// 2. Распарсить JSON из r.Body. Если ошибка -> 400 Bad Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "error decode json", http.StatusBadRequest)
			return
		}
		err := transferrer.Transfer(r.Context(), req.FromWalletID, req.ToWalletID, req.Amount)
		if err != nil {
			if errors.Is(err, postgresql.ErrInsufficientFunds) {
				http.Error(w, "insufficient funds", http.StatusBadRequest)
				return
			} else {
				log.Error("failed to transfer", slog.String("error", err.Error()))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)

		// 3. Вызвать бизнес-логику: transferrer.Transfer(...)
		// 4. Если ошибка:
		//    - Если это ошибка "insufficient funds" (недостаточно средств) -> 400 Bad Request
		//    - Иначе -> логируем ошибку и отдаем 500 Internal Server Error

		// 5. Если всё ок -> вернуть статус 200 OK
		// (Тело ответа можно оставить пустым или вернуть {"status": "ok"})
	}
}
