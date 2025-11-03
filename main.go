package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/IdrisovMarat/httpserver/internal/database"
	"github.com/IdrisovMarat/httpserver/internal/handlers"
	"github.com/IdrisovMarat/httpserver/internal/helpers"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	platform := os.Getenv("PLATFORM")
	polkaKey := os.Getenv("POLKA_KEY")

	if platform == "" {
		platform = "production" // default to production for safety
	}

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET не установлен в .env файле")
	}

	if polkaKey == "" {
		log.Fatal("POLKA_KEY не установлен в .env файле")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Something went wrong")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to database: %v\n", err)
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()

	config := &handlers.ApiConfig{
		Db:        dbQueries,
		Platform:  platform,
		JWTsecret: jwtSecret,
		PolkaKey:  polkaKey,
	}

	chainMiddlwareLog := func(h http.Handler) http.Handler {
		return helpers.MiddlewareLog(helpers.MiddlewareRecovery(h))
	}

	server := &http.Server{
		Addr:         ":" + helpers.ServerPort,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}

	// Используем структуру как http.Handler
	readyHandler := handlers.ReadyHandler{}
	// Файловый сервер
	fileServer := http.FileServer(http.Dir("."))
	assetsServer := http.FileServer(http.Dir("./assets"))

	mux.Handle("GET /api/healthz", chainMiddlwareLog(readyHandler))
	mux.Handle("GET /app/", chainMiddlwareLog(config.MiddlewareMetricsInt(http.StripPrefix("/app", fileServer))))
	mux.Handle("GET /assets/", chainMiddlwareLog(http.StripPrefix("/assets", assetsServer)))
	mux.HandleFunc("GET /admin/metrics", chainMiddlwareLog(http.HandlerFunc(config.MetricsHandler)).ServeHTTP)
	mux.HandleFunc("POST /admin/reset", chainMiddlwareLog(http.HandlerFunc(config.ResetmetricsHandler)).ServeHTTP)
	mux.HandleFunc("GET /api/debug/db", chainMiddlwareLog(http.HandlerFunc(config.DebugDBHandler)).ServeHTTP)

	mux.HandleFunc("POST /api/users", chainMiddlwareLog(http.HandlerFunc(config.CreateUserHandler)).ServeHTTP)
	mux.HandleFunc("POST /api/login", chainMiddlwareLog(http.HandlerFunc(config.LoginHandler)).ServeHTTP)
	mux.HandleFunc("PUT /api/users", chainMiddlwareLog(http.HandlerFunc(config.UpdateUserHandler)).ServeHTTP)

	mux.HandleFunc("POST /api/chirps", chainMiddlwareLog(http.HandlerFunc(config.CreateChirpHandler)).ServeHTTP)
	mux.HandleFunc("GET /api/chirps", chainMiddlwareLog(http.HandlerFunc(config.GetChirpsHandler)).ServeHTTP)
	mux.HandleFunc("GET /api/chirps/{chirpID}", chainMiddlwareLog(http.HandlerFunc(config.GetChirpByIdHandler)).ServeHTTP)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", chainMiddlwareLog(http.HandlerFunc(config.DeleteChirpHandler)).ServeHTTP)

	mux.HandleFunc("POST /api/refresh", chainMiddlwareLog(http.HandlerFunc(config.RefreshTokenHandler)).ServeHTTP)
	mux.HandleFunc("POST /api/revoke", chainMiddlwareLog(http.HandlerFunc(config.RevokeTokenHandler)).ServeHTTP)

	mux.HandleFunc("POST /api/polka/webhooks", chainMiddlwareLog(http.HandlerFunc(config.PolkaWebhookHandler)).ServeHTTP) // вебхуки

	log.Printf("🚀 HTTP сервер запущен на порту %s", server.Addr)
	fmt.Printf("📚 Документация API Chirpy:\n")
	fmt.Printf("\n🔐 Аутентификация:\n")
	fmt.Printf("   POST /api/users        - регистрация нового пользователя\n")
	fmt.Printf("   POST /api/login        - вход пользователя (возвращает access и refresh токены)\n")
	fmt.Printf("   POST /api/refresh      - обновление access токена\n")
	fmt.Printf("   POST /api/revoke       - отзыв refresh токена\n")
	fmt.Printf("   PUT  /api/users        - обновление данных пользователя\n")

	fmt.Printf("\n🐦 Chirps:\n")
	fmt.Printf("   POST /api/chirps       - создание нового chirp (требует аутентификации)\n")
	fmt.Printf("   GET  /api/chirps       - получение всех chirps (опционально: ?author_id=UUID&sort=asc|desc)\n")
	fmt.Printf("   GET  /api/chirps/{id}  - получение chirp по ID\n")
	fmt.Printf("   DELETE /api/chirps/{id} - удаление chirp (только автор)\n")

	fmt.Printf("\n⚙️  Администрирование:\n")
	fmt.Printf("   GET  /admin/metrics    - просмотр метрик\n")
	fmt.Printf("   POST /admin/reset      - сброс метрик (только в dev режиме)\n")

	fmt.Printf("\n🌐 Вебхуки:\n")
	fmt.Printf("   POST /api/polka/webhooks - обработка вебхуков от Polka (требует API ключ)\n")

	fmt.Printf("\n📋 Примеры использования:\n")
	fmt.Printf("   Регистрация: curl -X POST http://localhost:8080/api/users -d '{\"email\":\"user@example.com\",\"password\":\"pass\"}'\n")
	fmt.Printf("   Получение chirps автора: curl http://localhost:8080/api/chirps?author_id=UUID\n")
	fmt.Printf("   Создание chirp: curl -X POST -H 'Authorization: Bearer TOKEN' http://localhost:8080/api/chirps -d '{\"body\":\"Text\"}'\n")
	fmt.Printf("   Получение chirps с сортировкой: curl http://localhost:8080/api/chirps?sort=desc\n")
	fmt.Printf("   Фильтрация и сортировка: curl http://localhost:8080/api/chirps?author_id=UUID&sort=desc\n")

	fmt.Printf("\n------------------------------------------------------------------------------------------------------------------------------------\n")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Ошибка запуска сервера: %v", err)
	}
}
