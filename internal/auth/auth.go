package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// MakeRefreshToken создает криптографически безопасный refresh token
// Production рекомендация: используем crypto/rand вместо math/rand для безопасности
func MakeRefreshToken() (string, error) {
	// 32 bytes = 256 bits (достаточно для безопасности)
	bytes := make([]byte, 32)

	// Production: crypto/rand криптографически безопасен
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("ошибка генерации случайных байт: %w", err)
	}

	// Hex encoding для читаемости и безопасности
	return hex.EncodeToString(bytes), nil
}

// HashPassword хеширует пароль с использованием Argon2id
func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// CheckPasswordHash проверяет пароль с хешем
func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

// MakeJWT создает JWT токен для пользователя
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	// Создаем claims (данные токена)
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}

	// Создаем токен с claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен секретным ключом
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("ошибка подписи токена: %w", err)
	}

	return signedToken, nil
}

// ValidateJWT проверяет и валидирует JWT токен
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// Парсим токен с claims
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("ошибка парсинга токена: %w", err)
	}

	// Проверяем валидность токена
	if !token.Valid {
		return uuid.Nil, fmt.Errorf("невалидный токен")
	}

	// Извлекаем claims
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("неверный формат claims")
	}

	// Проверяем issuer
	if claims.Issuer != "chirpy" {
		return uuid.Nil, fmt.Errorf("неверный issuer")
	}

	// Извлекаем user ID из subject
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("неверный формат user ID: %w", err)
	}

	return userID, nil
}

// GetBearerToken извлекает токен из заголовка Authorization
func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("отсутствует заголовок Authorization")
	}

	// Проверяем формат "Bearer TOKEN"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("неверный формат заголовка Authorization")
	}

	return parts[1], nil
}

// GetAPIKey извлекает API ключ из заголовка Authorization
func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("отсутствует заголовок Authorization")
	}

	// Проверяем формат "ApiKey THE_KEY_HERE"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "apikey" {
		return "", fmt.Errorf("неверный формат заголовка Authorization")
	}

	return parts[1], nil
}
