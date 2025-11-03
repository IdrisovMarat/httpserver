package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/IdrisovMarat/httpserver/internal/auth"
	"github.com/IdrisovMarat/httpserver/internal/database"
	"github.com/IdrisovMarat/httpserver/internal/helpers"
	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func (cfg *ApiConfig) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	reqBody := requestBody{}
	err := decoder.Decode(&reqBody)
	if err != nil {
		log.Printf("❌ Ошибка декодирования JSON: %v", err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	// Проверяем email
	if reqBody.Email == "" {
		helpers.RespondWithError(w, http.StatusBadRequest, "Email обязателен")
		return
	}

	if reqBody.Password == "" {
		helpers.RespondWithError(w, http.StatusBadRequest, "Пароль обязателен")
		return
	}

	log.Printf("🔄 Попытка создать пользователя с email: %s", reqBody.Email)

	// Хешируем пароль
	hashedPassword, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		log.Printf("❌ Ошибка хеширования пароля: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать пользователя")
		return
	}

	userParam := database.CreateUserParams{
		Email:          reqBody.Email,
		HashedPassword: hashedPassword,
	}

	// Создаем пользователя в базе
	dbUser, err := cfg.Db.CreateUser(r.Context(), userParam)
	if err != nil {
		log.Printf("❌ Ошибка создания пользователя в БД: %v", err)
		// Проверяем нарушение уникальности email
		if strings.Contains(err.Error(), "unique") {
			helpers.RespondWithError(w, http.StatusConflict, "Email уже существует")
			return
		}
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать пользователя")
		return
	}

	log.Printf("✅ Пользователь создан успешно. ID: %s", dbUser.ID)

	// Конвертируем пользователя из БД в API формат
	user := User{
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}

	helpers.RespondWithJSON(w, http.StatusCreated, user)
}

func (cfg *ApiConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		User
		Token        string `json:"token"`         // Access token (JWT)
		RefreshToken string `json:"refresh_token"` // Refresh token
	}

	decoder := json.NewDecoder(r.Body)
	reqBody := requestBody{}
	err := decoder.Decode(&reqBody)
	if err != nil {
		log.Printf("❌ Ошибка декодирования JSON: %v", err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	// Проверяем email и пароль
	if reqBody.Email == "" || reqBody.Password == "" {
		helpers.RespondWithError(w, http.StatusBadRequest, "Email и пароль обязательны")
		return
	}

	// Production: Проверка длины email для предотвращения атак
	if len(reqBody.Email) > 255 {
		helpers.RespondWithError(w, http.StatusBadRequest, "Email слишком длинный")
		return
	}

	log.Printf("🔄 Попытка входа пользователя: %s", reqBody.Email)

	// Ищем пользователя по email
	dbUser, err := cfg.Db.GetUserByEmail(r.Context(), reqBody.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Пользователь с email %s не найден", reqBody.Email)
			helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный email или пароль")
			return
		}
		log.Printf("❌ Ошибка поиска пользователя: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	// Проверяем пароль
	match, err := auth.CheckPasswordHash(reqBody.Password, dbUser.HashedPassword)
	if err != nil {
		log.Printf("❌ Ошибка проверки пароля: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	if !match {
		log.Printf("❌ Неверный пароль для пользователя: %s", reqBody.Email)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный email или пароль")
		return
	}

	// Создаем JWT токен
	token, err := auth.MakeJWT(dbUser.ID, cfg.JWTsecret, time.Hour)
	if err != nil {
		log.Printf("❌ Ошибка создания JWT токена: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать токен")
		return
	}

	// Создаем refresh token (60 дней)
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("❌ Ошибка создания refresh token: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать токен")
		return
	}

	// Сохраняем refresh token в базе
	_, err = cfg.Db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    dbUser.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour), // 60 дней
	})
	if err != nil {
		log.Printf("❌ Ошибка сохранения refresh token: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать токен")
		return
	}

	log.Printf("✅ Успешный вход пользователя: %s", dbUser.ID)
	// Production: Логируем успешный вход для аудита
	log.Printf("🔐 Создан access token (1h) и refresh token (60d) для пользователя: %s", dbUser.ID)

	// Возвращаем пользователя без пароля
	resp := response{
		User: User{
			ID:          dbUser.ID,
			CreatedAt:   dbUser.CreatedAt,
			UpdatedAt:   dbUser.UpdatedAt,
			Email:       dbUser.Email,
			IsChirpyRed: dbUser.IsChirpyRed,
		},
		Token:        token,
		RefreshToken: refreshToken,
	}

	helpers.RespondWithJSON(w, http.StatusOK, resp)
}

func (cfg *ApiConfig) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	// 🔐 АУТЕНТИФИКАЦИЯ: Проверяем access token
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("❌ Ошибка извлечения токена: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или отсутствующий токен")
		return
	}

	// 🔐 АУТЕНТИФИКАЦИЯ: Валидируем JWT токен
	userID, err := auth.ValidateJWT(tokenString, cfg.JWTsecret)
	if err != nil {
		log.Printf("❌ Ошибка валидации токена: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный токен")
		return
	}

	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	reqBody := requestBody{}
	err = decoder.Decode(&reqBody)
	if err != nil {
		log.Printf("❌ Ошибка декодирования JSON: %v", err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	// 🛡️ ВАЛИДАЦИЯ: Проверяем что хотя бы одно поле заполнено
	if reqBody.Email == "" && reqBody.Password == "" {
		helpers.RespondWithError(w, http.StatusBadRequest, "Необходимо указать email или пароль для обновления")
		return
	}

	// 🛡️ ВАЛИДАЦИЯ: Проверяем длину email
	if reqBody.Email != "" && len(reqBody.Email) > 255 {
		helpers.RespondWithError(w, http.StatusBadRequest, "Email слишком длинный")
		return
	}

	log.Printf("🔄 Попытка обновления пользователя: %s", userID)

	// Подготавливаем данные для обновления
	updateParams := database.UpdateUserParams{
		ID: userID, // 🔐 АВТОРИЗАЦИЯ: Обновляем только текущего пользователя
	}

	// Если указан email - обновляем его
	if reqBody.Email != "" {
		updateParams.Email = reqBody.Email
	} else {
		// Если email не указан, получаем текущий email из БД
		currentUser, err := cfg.Db.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("❌ Ошибка получения текущего пользователя: %v", err)
			helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
			return
		}
		updateParams.Email = currentUser.Email
	}

	// Если указан пароль - хешируем и обновляем
	if reqBody.Password != "" {
		// 🛡️ БЕЗОПАСНОСТЬ: Хешируем пароль перед сохранением
		hashedPassword, err := auth.HashPassword(reqBody.Password)
		if err != nil {
			log.Printf("❌ Ошибка хеширования пароля: %v", err)
			helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось обновить пользователя")
			return
		}
		updateParams.HashedPassword = hashedPassword
	} else {
		// Если пароль не указан, получаем текущий хеш из БД
		currentUser, err := cfg.Db.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("❌ Ошибка получения текущего пользователя: %v", err)
			helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
			return
		}
		updateParams.HashedPassword = currentUser.HashedPassword
	}

	// 💾 ОБНОВЛЕНИЕ В БАЗЕ
	updatedUser, err := cfg.Db.UpdateUser(r.Context(), updateParams)
	if err != nil {
		log.Printf("❌ Ошибка обновления пользователя в БД: %v", err)

		// 🔐 АВТОРИЗАЦИЯ: Проверяем нарушение уникальности email
		if strings.Contains(err.Error(), "unique") {
			helpers.RespondWithError(w, http.StatusConflict, "Email уже используется другим пользователем")
			return
		}

		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось обновить пользователя")
		return
	}

	// 🛡️ БЕЗОПАСНОСТЬ: Принудительно отзываем все refresh tokens при смене пароля
	if reqBody.Password != "" {
		err = cfg.Db.RevokeAllUserRefreshTokens(r.Context(), userID)
		if err != nil {
			log.Printf("⚠️ Ошибка отзыва refresh tokens: %v", err)
			// Не прерываем выполнение, только логируем
		}
		log.Printf("🔐 Отозваны все refresh tokens пользователя %s из-за смены пароля", userID)
	}

	log.Printf("✅ Пользователь успешно обновлен: %s", userID)

	// 📤 ОТВЕТ: Возвращаем обновленного пользователя (без пароля)
	response := User{
		ID:          updatedUser.ID,
		CreatedAt:   updatedUser.CreatedAt,
		UpdatedAt:   updatedUser.UpdatedAt,
		Email:       updatedUser.Email,
		IsChirpyRed: updatedUser.IsChirpyRed,
	}

	helpers.RespondWithJSON(w, http.StatusOK, response)
}
