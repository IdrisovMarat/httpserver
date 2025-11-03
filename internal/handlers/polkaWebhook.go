package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/IdrisovMarat/httpserver/internal/auth"
	"github.com/IdrisovMarat/httpserver/internal/helpers"
	"github.com/google/uuid"
)

// WebhookRequest представляет структуру вебхука от Polka
type WebhookRequest struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *ApiConfig) PolkaWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// 🔐 АУТЕНТИФИКАЦИЯ: Проверяем API ключ Polka
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		log.Printf("❌ Ошибка извлечения API ключа: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или отсутствующий API ключ")
		return
	}

	// 🔐 АУТЕНТИФИКАЦИЯ: Проверяем что ключ совпадает
	if apiKey != cfg.PolkaKey {
		log.Printf("🚫 Неверный API ключ: получен %s, ожидался %s", apiKey, cfg.PolkaKey)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный API ключ")
		return
	}
	// Определяем структуру для тела запроса вебхука
	type requestBody struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	// Декодируем JSON из тела запроса
	decoder := json.NewDecoder(r.Body)
	reqBody := requestBody{}
	err = decoder.Decode(&reqBody)
	if err != nil {
		log.Printf("❌ Ошибка декодирования JSON вебхука: %v", err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	// Проверяем обязательные поля
	if reqBody.Event == "" {
		log.Printf("❌ Отсутствует поле event в вебхуке")
		helpers.RespondWithError(w, http.StatusBadRequest, "Поле event обязательно")
		return
	}

	if reqBody.Data.UserID == "" {
		log.Printf("❌ Отсутствует поле data.user_id в вебхуке")
		helpers.RespondWithError(w, http.StatusBadRequest, "Поле data.user_id обязательно")
		return
	}

	log.Printf("🔄 Получен вебхук от Polka: событие '%s' для пользователя %s", reqBody.Event, reqBody.Data.UserID)

	// Обрабатываем только событие user.upgraded
	if reqBody.Event != "user.upgraded" {
		log.Printf("ℹ️  Игнорируем неизвестное событие: %s", reqBody.Event)
		// Возвращаем 204 для неизвестных событий (требование задания)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Парсим ID пользователя
	userID, err := uuid.Parse(reqBody.Data.UserID)
	if err != nil {
		log.Printf("❌ Неверный формат UUID пользователя: %s, ошибка: %v", reqBody.Data.UserID, err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат ID пользователя")
		return
	}

	log.Printf("🔄 Обработка апгрейда пользователя до Chirpy Red: %s", userID)

	// Проверяем существование пользователя
	_, err = cfg.Db.GetUserByID(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Пользователь с ID %s не найден", userID)
			helpers.RespondWithError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}
		log.Printf("❌ Ошибка поиска пользователя в БД: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	// Обновляем пользователя до Chirpy Red
	err = cfg.Db.UpgradeUserToChirpyRed(r.Context(), userID)
	if err != nil {
		log.Printf("❌ Ошибка обновления пользователя до Chirpy Red: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось обновить пользователя")
		return
	}

	log.Printf("✅ Пользователь %s успешно обновлен до Chirpy Red", userID)

	// Возвращаем 204 No Content при успешном обновлении
	w.WriteHeader(http.StatusNoContent)
}
