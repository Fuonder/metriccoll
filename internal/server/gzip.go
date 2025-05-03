// Package server содержит реализацию GZIP-сжатия и разжатия данных для HTTP-запросов и ответов.
// gzip.go реализует обертки для http.ResponseWriter и io.ReadCloser, обеспечивая прозрачную работу с GZIP.
package server

import (
	"compress/gzip"
	"io"
	"net/http"
)

// gzipWriter — обертка над http.ResponseWriter, которая обеспечивает
// автоматическое сжатие тела ответа с использованием алгоритма GZIP.
type gzipWriter struct {
	w  http.ResponseWriter // оригинальный http.ResponseWriter
	zw *gzip.Writer        // GZIP-обертка для записи
}

// newGzipWriter создает новый gzipWriter, который будет сжимать данные перед отправкой клиенту.
func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header возвращает заголовки HTTP-ответа.
// Необходим для соответствия интерфейсу http.ResponseWriter.
func (c *gzipWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает сжатые данные в поток ответа.
// Используется для отправки тела ответа клиенту.
func (c *gzipWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteHeader устанавливает HTTP-статус и заголовок Content-Encoding: gzip,
// если статус меньше 300 (успешный).
func (c *gzipWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close завершает работу gzip.Writer и освобождает ресурсы.
func (c *gzipWriter) Close() error {
	return c.zw.Close()
}

// gzipReader — обертка над io.ReadCloser, которая обеспечивает
// автоматическое разжатие входящего тела запроса, если оно было сжато с помощью GZIP.
type gzipReader struct {
	r  io.ReadCloser // оригинальное тело запроса
	zr *gzip.Reader  // GZIP-обертка для чтения
}

// newGzipReader создает новый gzipReader для чтения и разжатия тела запроса.
func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read считывает разжатые данные из тела запроса.
func (c gzipReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close закрывает как исходный io.ReadCloser, так и gzip.Reader.
func (c *gzipReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
