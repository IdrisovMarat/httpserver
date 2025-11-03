package helpers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	ContentTypePlain = "text/plain; charset=utf-8"
	ContentTypeHTML  = "text/html; charset=utf-8"
	ContentTypeJSON  = "application/json"
	ServerPort       = "8080"
)

func MiddlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// middlewareRecovery - восстановление после паник
func MiddlewareRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("🚨 Паника восстановлена: %v", err)
				http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func DelProfanWords(str string) string {
	profanWords := [3]string{"kerfuffle", "sharbert", "fornax"}

	slicewords := strings.Split(str, " ")

	newslice := make([]string, 0)

	flag := true

	for _, word := range slicewords {
		flag = false
		for _, profan := range profanWords {
			if strings.ToLower(word) == profan {
				flag = true
				break
			}
		}

		if flag {
			newslice = append(newslice, "****")
			continue
		}

		newslice = append(newslice, word)
	}

	return strings.Join(newslice, " ")
}

// Helper function to send JSON responses
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Helper function to send error responses
func RespondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	RespondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}
