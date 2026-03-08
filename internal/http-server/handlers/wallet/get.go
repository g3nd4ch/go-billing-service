package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"myApi/internal/storage/postgresql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type WalletGetter interface {
	GetWallet(ctx context.Context, walletID string) (*postgresql.Wallet, error)
}

func Get(log *slog.Logger, getter WalletGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ТВОЯ ЗАДАЧА: Написать логику здесь
		id := chi.URLParam(r, "id")
		// 1. Достать ID кошелька из URL.
		// В роутере chi это делается так:
		// id := chi.URLParam(r, "id")

		// 2. Сделать валидацию: если id пустой (id == ""), вернуть статус 400 и ошибку.
		if id == "" {
			http.Error(w, "id is empty", http.StatusBadRequest)
			return
		}

		// 3. Вызвать бизнес-логику: getter.GetWallet(r.Context(), id)
		wallet, err := getter.GetWallet(r.Context(), id)
		if err != nil {
			if errors.Is(err, postgresql.ErrWalletNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			} else {
				log.Error("failed to get wallet", slog.String("error", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(wallet); err != nil{
			http.Error(w, "error encode json", http.StatusInternalServerError)
			return
		}
		// 4. Обработать ошибку.
		// Если ошибка есть, проверь её:
		// Если это ошибка "wallet not found" -> верни статус 404 (Not Found).
		// Иначе -> залогируй и верни статус 500 (Internal Server Error).

		// 5. Если всё ок:
		// Установить заголовок Content-Type: application/json
		// Вернуть JSON с данными кошелька (json.NewEncoder(w).Encode(wallet))
	}
}
