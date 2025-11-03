package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/IdrisovMarat/httpserver/internal/helpers"
)

func (cfg *ApiConfig) MiddlewareMetricsInt(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1) // ✅ Теперь счетчик увеличивается при каждом запросе
		next.ServeHTTP(w, r)
	})
}

// ReadyHandler структура для обработчика
type ReadyHandler struct{}

// ServeHTTP делает ReadyHandler http.Handler
func (h ReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", helpers.ContentTypePlain)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Ошибка записи ответа /healthz: %v", err)
	}
}

func (cfg *ApiConfig) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.FileserverHits.Load()
	w.Header().Set("Content-Type", helpers.ContentTypeHTML)
	htmlTemplate := `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

	htmlContent := fmt.Sprintf(htmlTemplate, hits)

	if _, err := w.Write([]byte(htmlContent)); err != nil {
		log.Printf("Ошибка записи ответа /metrics: %v", err)
	}
}

func (cfg *ApiConfig) ResetmetricsHandler(w http.ResponseWriter, r *http.Request) {
	cfg.FileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("Metrics reset")); err != nil {
		log.Printf("Ошибка записи ответа /reset: %v", err)
	}
}

func (cfg *ApiConfig) DebugDBHandler(w http.ResponseWriter, r *http.Request) {
	// Простой запрос для проверки подключения к БД
	err := cfg.Db.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("❌ Ошибка подключения к БД: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Ошибка БД: %v", err))
		return
	}

	err = cfg.Db.DeleteAllChirps(r.Context())
	if err != nil {
		log.Printf("❌ Ошибка подключения к БД: %v", err)
		helpers.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Ошибка БД: %v", err))
		return
	}

	helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "База данных подключена и очищена"})
}
