package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"net/http"
)

type hashWriter struct {
	w   http.ResponseWriter
	key string
}

func newHashWriter(w http.ResponseWriter, key string) *hashWriter {
	return &hashWriter{
		w:   w,
		key: key,
	}
}

func (hw *hashWriter) Header() http.Header {
	return hw.w.Header()
}

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

func (hw *hashWriter) WriteHeader(statusCode int) {
	hw.w.WriteHeader(statusCode)
}

func calculateHMAC(body []byte, key string) (string, error) {
	hm := hmac.New(sha256.New, []byte(key))
	hm.Write(body)
	s := hm.Sum(nil)
	return base64.URLEncoding.EncodeToString(s), nil
}

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
