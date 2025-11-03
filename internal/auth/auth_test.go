package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("HashPassword returned empty hash")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "mysecretpassword"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Проверка правильного пароля
	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}
	if !match {
		t.Error("CheckPasswordHash should return true for correct password")
	}

	// Проверка неправильного пароля
	match, err = CheckPasswordHash(wrongPassword, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}
	if match {
		t.Error("CheckPasswordHash should return false for wrong password")
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	if token == "" {
		t.Error("MakeJWT returned empty token")
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Hour

	// Создаем валидный токен
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	// Проверяем валидный токен
	validatedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("ValidateJWT failed for valid token: %v", err)
	}

	if validatedUserID != userID {
		t.Errorf("ValidateJWT returned wrong user ID: got %v, want %v", validatedUserID, userID)
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"

	// Создаем токен с истекшим сроком
	expiresIn := -time.Hour // отрицательное время = токен уже истек
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	// Пытаемся проверить истекший токен
	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Error("ValidateJWT should fail for expired token")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	wrongSecret := "wrong-secret"
	expiresIn := time.Hour

	// Создаем токен с правильным секретом
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	// Пытаемся проверить с неправильным секретом
	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Error("ValidateJWT should fail for token signed with wrong secret")
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	tokenSecret := "test-secret"

	// Пытаемся проверить невалидный токен
	_, err := ValidateJWT("invalid-token-string", tokenSecret)
	if err == nil {
		t.Error("ValidateJWT should fail for invalid token string")
	}
}

func TestValidateJWT_WrongIssuer(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Hour

	// Вручную создаем токен с неправильным issuer
	claims := jwt.RegisteredClaims{
		Issuer:    "wrong-issuer", // неправильный issuer
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		t.Fatalf("Failed to create token with wrong issuer: %v", err)
	}

	// Пытаемся проверить токен с неправильным issuer
	_, err = ValidateJWT(tokenString, tokenSecret)
	if err == nil {
		t.Error("ValidateJWT should fail for token with wrong issuer")
	}
}

func TestGetBearerToken(t *testing.T) {
	// Тест с валидным заголовком
	headers := http.Header{}
	headers.Set("Authorization", "Bearer valid-token-123")

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("GetBearerToken failed for valid header: %v", err)
	}

	if token != "valid-token-123" {
		t.Errorf("GetBearerToken returned wrong token: got %s, want %s", token, "valid-token-123")
	}
}

func TestGetBearerToken_NoHeader(t *testing.T) {
	// Тест без заголовка Authorization
	headers := http.Header{}

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error("GetBearerToken should fail when Authorization header is missing")
	}
}

func TestGetBearerToken_InvalidFormat(t *testing.T) {
	// Тест с неверным форматом
	headers := http.Header{}
	headers.Set("Authorization", "InvalidFormat token")

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error("GetBearerToken should fail for invalid header format")
	}
}

func TestGetBearerToken_NoBearer(t *testing.T) {
	// Тест без префикса Bearer
	headers := http.Header{}
	headers.Set("Authorization", "Token valid-token")

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error("GetBearerToken should fail when Bearer prefix is missing")
	}
}

func TestGetAPIKey(t *testing.T) {
	// Тест с валидным заголовком
	headers := http.Header{}
	headers.Set("Authorization", "ApiKey valid-api-key-123")

	apiKey, err := GetAPIKey(headers)
	if err != nil {
		t.Fatalf("GetAPIKey failed for valid header: %v", err)
	}

	if apiKey != "valid-api-key-123" {
		t.Errorf("GetAPIKey returned wrong API key: got %s, want %s", apiKey, "valid-api-key-123")
	}
}

func TestGetAPIKey_NoHeader(t *testing.T) {
	// Тест без заголовка Authorization
	headers := http.Header{}

	_, err := GetAPIKey(headers)
	if err == nil {
		t.Error("GetAPIKey should fail when Authorization header is missing")
	}
}

func TestGetAPIKey_InvalidFormat(t *testing.T) {
	// Тест с неверным форматом
	headers := http.Header{}
	headers.Set("Authorization", "InvalidFormat key")

	_, err := GetAPIKey(headers)
	if err == nil {
		t.Error("GetAPIKey should fail for invalid header format")
	}
}

func TestGetAPIKey_NoApiKeyPrefix(t *testing.T) {
	// Тест без префикса ApiKey
	headers := http.Header{}
	headers.Set("Authorization", "Bearer some-token")

	_, err := GetAPIKey(headers)
	if err == nil {
		t.Error("GetAPIKey should fail when ApiKey prefix is missing")
	}
}
