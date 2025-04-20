// Package server реализует создание и проверку HMAC-подписи HTTP-сообщений с использованием SHA-256.
// hmacSHA256.go содержит обертку для ResponseWriter и функции генерации и валидации подписи.
package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/Fuonder/metriccoll.git/internal/logger"
)

// hashWriter — обертка над http.ResponseWriter, добавляющая HMAC-SHA256-подпись в заголовок ответа.
type hashWriter struct {
	w   http.ResponseWriter
	key string
}

// newHashWriter создает новый hashWriter с заданным HMAC-ключом.
func newHashWriter(w http.ResponseWriter, key string) *hashWriter {
	return &hashWriter{
		w:   w,
		key: key,
	}
}

// Header возвращает заголовки HTTP-ответа.
// Необходим для соответствия интерфейсу http.ResponseWriter.
func (hw *hashWriter) Header() http.Header {
	return hw.w.Header()
}

// Write записывает данные в тело ответа и добавляет HMAC-подпись в заголовок `HashSHA256`.
func (hw *hashWriter) Write(p []byte) (int, error) {
	if hw.key != "" {
		res, err := calculateHMAC(p, hw.key)
		if err != nil {
			return 0, err
		}
		logger.Log.Info("Writing HMAC to response")
		hw.w.Header().Set("HashSHA256", res)
	}
	return hw.w.Write(p)
}

// WriteHeader передает статусный код в оригинальный ResponseWriter.
func (hw *hashWriter) WriteHeader(statusCode int) {
	hw.w.WriteHeader(statusCode)
}

// calculateHMAC вычисляет HMAC-SHA256 для заданного тела сообщения и ключа.
// Возвращает HMAC в виде строки, закодированной в base64.
func calculateHMAC(body []byte, key string) (string, error) {
	hm := hmac.New(sha256.New, []byte(key))
	hm.Write(body)
	s := hm.Sum(nil)
	return base64.URLEncoding.EncodeToString(s), nil
}

// validateHMAC проверяет, совпадает ли HMAC-подпись из запроса с вычисленным значением.
// Возвращает ErrMismatchedHash в случае несовпадения.
func validateHMAC(packetHash string, body []byte, key string) error {
	calculatedHash, err := calculateHMAC(body, key)
	if err != nil {
		return err
	}
	if packetHash != calculatedHash {
		return ErrMismatchedHash
	}
	return nil
}
