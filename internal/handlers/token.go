package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/IdrisovMarat/httpserver/internal/auth"
	"github.com/IdrisovMarat/httpserver/internal/helpers"
)

func (cfg *ApiConfig) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"` // Новый access token
	}

	// Извлекаем refresh token из заголовка
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("❌ Ошибка извлечения refresh token: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или отсутствующий токен")
		return
	}

	// Production: Проверяем формат токена (должен быть 64 hex символа)
	if len(tokenString) != 64 {
		log.Printf("❌ Неверный формат refresh token")
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный токен")
		return
	}

	log.Printf("🔄 Попытка обновления токена с refresh token: %s...", tokenString[:8])

	// Ищем пользователя по валидному refresh token
	dbUser, err := cfg.Db.GetUserFromRefreshToken(r.Context(), tokenString)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Refresh token не найден или невалиден: %s...", tokenString[:8])
			helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или истекший токен")
			return
		}
		log.Printf("❌ Ошибка поиска refresh token: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	// Создаем новый access token
	accessToken, err := auth.MakeJWT(dbUser.ID, cfg.JWTsecret, time.Hour)
	if err != nil {
		log.Printf("❌ Ошибка создания access token: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать токен")
		return
	}

	log.Printf("✅ Успешное обновление access token для пользователя: %s", dbUser.ID)

	// Production: Логируем обновление токена для аудита
	log.Printf("🔄 Выдан новый access token для пользователя: %s", dbUser.ID)

	resp := response{
		Token: accessToken,
	}

	helpers.RespondWithJSON(w, http.StatusOK, resp)
}

func (cfg *ApiConfig) RevokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем refresh token из заголовка
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("❌ Ошибка извлечения refresh token для отзыва: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или отсутствующий токен")
		return
	}

	// Production: Проверяем формат токена
	if len(tokenString) != 64 {
		log.Printf("❌ Неверный формат refresh token при отзыве")
		w.WriteHeader(http.StatusNoContent) // 204 - без тела
		return
	}

	log.Printf("🔄 Попытка отзыва refresh token: %s...", tokenString[:8])

	// Отзываем токен в базе
	err = cfg.Db.RevokeRefreshToken(r.Context(), tokenString)
	if err != nil {
		if err == sql.ErrNoRows {
			// Production: Даже если токен не найден, возвращаем 204 для безопасности
			log.Printf("⚠️ Refresh token не найден при отзыве: %s...", tokenString[:8])
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Printf("❌ Ошибка отзыва refresh token: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	log.Printf("✅ Успешный отзыв refresh token: %s...", tokenString[:8])

	// Production: 204 No Content - успешно, но без тела ответа
	w.WriteHeader(http.StatusNoContent)
}
