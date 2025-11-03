package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/IdrisovMarat/httpserver/internal/auth"
	"github.com/IdrisovMarat/httpserver/internal/database"
	"github.com/IdrisovMarat/httpserver/internal/helpers"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// sortChirps сортирует chirps в соответствии с указанным порядком
func SortChirps(chirps []Chirp, sortOrder string) []Chirp {
	switch sortOrder {
	case "desc":
		// Сортировка по убыванию (новые сначала)
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt > chirps[j].CreatedAt
		})
		log.Printf("📊 Применена сортировка по убыванию (desc)")
	case "asc":
		// Сортировка по возрастанию (старые сначала) - значение по умолчанию
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt < chirps[j].CreatedAt
		})
		log.Printf("📊 Применена сортировка по возрастанию (asc)")
	default:
		// По умолчанию - сортировка по возрастанию
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt < chirps[j].CreatedAt
		})
		log.Printf("📊 Применена сортировка по умолчанию (asc)")
	}

	return chirps
}

func (cfg *ApiConfig) CreateChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем токен из заголовка
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("❌ Ошибка извлечения токена: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или отсутствующий токен")
		return
	}

	// Валидируем JWT токен
	userID, err := auth.ValidateJWT(tokenString, cfg.JWTsecret)
	if err != nil {
		log.Printf("❌ Ошибка валидации токена: %v", err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный токен")
		return
	}

	type chirpBody struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	chirp := chirpBody{}
	err = decoder.Decode(&chirp)
	if err != nil {
		log.Printf("❌ Ошибка декодирования JSON: %v", err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	if len(chirp.Body) >= 140 || len(chirp.Body) == 0 {
		helpers.RespondWithError(w, http.StatusBadRequest, "поле сhirp не может быть пустым и текс должен быть менее 140 символов")
		return
	}

	chirpParam := database.CreateChirpParams{
		Body:   helpers.DelProfanWords(chirp.Body),
		UserID: userID,
	}

	log.Printf("🔄 Попытка создать текст chirp: %s", chirpParam)

	// Создаем chirp в базе
	dbChirp, err := cfg.Db.CreateChirp(r.Context(), chirpParam)
	if err != nil {
		log.Printf("❌ Ошибка создания chirp в БД: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось создать chirp")
		return
	}

	log.Printf("✅ chirp создан успешно. ID: %s", dbChirp.ID)

	// Конвертируем chirp из БД в API формат
	respons := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: dbChirp.UpdatedAt.Format("2006-01-02 15:04:05"),
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	helpers.RespondWithJSON(w, http.StatusCreated, respons)
}

func (cfg *ApiConfig) GetChirpsHandler(w http.ResponseWriter, r *http.Request) {
	// 📋 Получаем параметр author_id из query string
	authorIDStr := r.URL.Query().Get("author_id")
	sortOrder := r.URL.Query().Get("sort")

	log.Printf("🔄 Получение chirps из базы данных, author_id: %s, sort: %s", authorIDStr, sortOrder)

	var dbChirps []database.Chirp
	var err error

	// 🔍 Если указан author_id - фильтруем по автору
	if authorIDStr != "" {
		// Парсим author_id в UUID
		authorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			log.Printf("❌ Неверный формат UUID author_id: %s, ошибка: %v", authorIDStr, err)
			helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат author_id")
			return
		}

		// Получаем chirps только для указанного автора
		dbChirps, err = cfg.Db.GetChirpsByAuthorID(r.Context(), authorID)
		if err != nil {
			log.Printf("❌ Ошибка получения chirps автора %s из БД: %v", authorID, err)
			helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось получить chirps")
			return
		}

		log.Printf("✅ Найдено %d chirps автора: %s", len(dbChirps), authorID)
	} else {
		// 📋 Если author_id не указан - получаем все chirps
		dbChirps, err = cfg.Db.GetChirps(r.Context())
		if err != nil {
			log.Printf("❌ Ошибка получения всех chirps из БД: %v", err)
			helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось получить chirps")
			return
		}

		log.Printf("✅ Найдено %d chirps (все)", len(dbChirps))
	}

	// Конвертируем chirps из БД в API формат
	chirps := make([]Chirp, len(dbChirps))
	for i, dbChirp := range dbChirps {
		chirps[i] = Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: dbChirp.UpdatedAt.Format(time.RFC3339Nano),
			// CreatedAt: dbChirp.CreatedAt.Format("2006-01-02 15:04:05"), // Формат: "2021-01-01 00:00:00"
			// UpdatedAt: dbChirp.UpdatedAt.Format("2006-01-02 15:04:05"),
			Body:   dbChirp.Body,
			UserID: dbChirp.UserID,
		}
	}

	// 🎯 Применяем сортировку
	chirps = SortChirps(chirps, sortOrder)

	helpers.RespondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *ApiConfig) GetChirpByIdHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем ID из пути
	chirpIDStr := r.PathValue("chirpID")

	if chirpIDStr == "" {
		log.Printf("❌ ID chirp не указан в пути")
		helpers.RespondWithError(w, http.StatusBadRequest, "ID chirp обязателен")
		return
	}

	// Парсим строку в UUID
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		log.Printf("❌ Неверный формат UUID: %s, ошибка: %v", chirpIDStr, err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат ID")
		return
	}

	log.Printf("🔄 Получение chirp с ID: %s из базы данных", chirpID)

	// Получаем chirp из базы данных
	dbChirp, err := cfg.Db.GetChirpsById(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Chirp с ID %s не найден", chirpID)
			helpers.RespondWithError(w, http.StatusNotFound, "Chirp не найден")
			return
		}
		log.Printf("❌ Ошибка получения chirp из БД: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось получить chirp")
		return
	}

	log.Printf("✅ Найден chirp ID: %s", dbChirp.ID)

	// Конвертируем chirp из БД в API формат
	response := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt.Format(time.RFC3339), // Формат: "2021-01-01T00:00:00Z"
		UpdatedAt: dbChirp.UpdatedAt.Format(time.RFC3339),
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	helpers.RespondWithJSON(w, http.StatusOK, response)
}

func (cfg *ApiConfig) DeleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	// 📍 Получаем ID chirp из пути ДО аутентификации (для логирования)
	chirpIDStr := r.PathValue("chirpID")
	if chirpIDStr == "" {
		log.Printf("❌ ID chirp не указан в пути")
		helpers.RespondWithError(w, http.StatusBadRequest, "ID chirp обязателен")
		return
	}

	// 🔄 Парсим chirp ID (валидация формата)
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		log.Printf("❌ Неверный формат UUID chirp: %s, ошибка: %v", chirpIDStr, err)
		helpers.RespondWithError(w, http.StatusBadRequest, "Неверный формат ID chirp")
		return
	}

	log.Printf("🔄 Попытка удаления chirp: %s", chirpID)

	// 🔐 АУТЕНТИФИКАЦИЯ: Проверяем access token
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("❌ Ошибка извлечения токена при удалении chirp %s: %v", chirpID, err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный или отсутствующий токен")
		return
	}

	// 🔐 АУТЕНТИФИКАЦИЯ: Валидируем JWT токен
	userID, err := auth.ValidateJWT(tokenString, cfg.JWTsecret)
	if err != nil {
		log.Printf("❌ Ошибка валидации токена при удалении chirp %s: %v", chirpID, err)
		helpers.RespondWithError(w, http.StatusUnauthorized, "Неверный токен")
		return
	}

	log.Printf("🔄 Пользователь %s пытается удалить chirp: %s", userID, chirpID)

	// 🔎 Находим chirp в базе данных
	dbChirp, err := cfg.Db.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Chirp с ID %s не найден, пользователь: %s", chirpID, userID)
			helpers.RespondWithError(w, http.StatusNotFound, "Chirp не найден")
			return
		}
		log.Printf("❌ Ошибка поиска chirp %s в БД, пользователь %s: %v", chirpID, userID, err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		return
	}

	// 🔐 АВТОРИЗАЦИЯ: Проверяем, что пользователь является автором chirp
	if dbChirp.UserID != userID {
		log.Printf("🚫 Попытка удаления чужого chirp. Chirp автор: %s, Пользователь: %s, Chirp ID: %s",
			dbChirp.UserID, userID, chirpID)

		// 🛡️ Production: Не раскрываем информацию о существовании chirp
		helpers.RespondWithError(w, http.StatusForbidden, "Недостаточно прав для выполнения этой операции")
		return
	}

	// 🗑️ Удаляем chirp из базы данных
	err = cfg.Db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		log.Printf("❌ Ошибка удаления chirp %s из БД, пользователь %s: %v", chirpID, userID, err)
		helpers.RespondWithError(w, http.StatusInternalServerError, "Не удалось удалить chirp")
		return
	}

	log.Printf("✅ Chirp успешно удален: %s пользователем: %s", chirpID, userID)

	// ✅ Возвращаем 204 No Content при успешном удалении
	w.WriteHeader(http.StatusNoContent)
	// Важно: НИКАКОГО тела ответа при 204 статусе!
}
