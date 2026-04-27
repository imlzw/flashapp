package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func hashPassword(password string) (string, error) {
	const iterations = 120000
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := stretchPassword(password, salt, iterations)
	return fmt.Sprintf("v1$%d$%s$%s", iterations, hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "v1" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := stretchPassword(password, salt, iterations)
	return subtle.ConstantTimeCompare(want, got) == 1
}

func stretchPassword(password string, salt []byte, iterations int) []byte {
	buf := make([]byte, 0, len(salt)+len(password))
	buf = append(buf, salt...)
	buf = append(buf, password...)
	sum := sha256.Sum256(buf)
	result := append([]byte(nil), sum[:]...)
	for i := 1; i < iterations; i++ {
		nextInput := make([]byte, 0, len(result)+len(salt))
		nextInput = append(nextInput, result...)
		nextInput = append(nextInput, salt...)
		next := sha256.Sum256(nextInput)
		result = append(result[:0], next[:]...)
	}
	return append([]byte(nil), result...)
}

func signToken(secret string, claims tokenClaims) (string, error) {
	headerBytes, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerBytes)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	message := header + "." + payload

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return message + "." + signature, nil
}

func parseToken(secret, token string) (tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, errors.New("invalid token format")
	}

	message := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	expected := mac.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return tokenClaims{}, err
	}
	if subtle.ConstantTimeCompare(expected, signature) != 1 {
		return tokenClaims{}, errors.New("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, err
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return tokenClaims{}, err
	}
	return claims, nil
}

func toUserView(item user) userView {
	return userView{
		ID:       item.ID,
		Username: item.Username,
		Nickname: item.Nickname,
	}
}

func normalizeUsername(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if len(value) < 3 || len(value) > 24 {
		return "", errors.New("username must be 3-24 characters")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", errors.New("username can only contain letters, numbers, underscore and hyphen")
	}
	return value, nil
}

func normalizeNickname(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	count := len([]rune(value))
	if count < 1 || count > 24 {
		return "", errors.New("nickname must be 1-24 characters")
	}
	return value, nil
}
