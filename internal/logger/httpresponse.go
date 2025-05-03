package logger

import "net/http"

// ResponseData содержит информацию о HTTP-ответе,
// такую как статус код, размер ответа, тип контента и кодировка контента.
type ResponseData struct {
	statusCode          int    // Статус код HTTP-ответа
	respSizeB           int    // Размер ответа в байтах
	respContentType     string // Тип контента HTTP-ответа
	respContentEncoding string // Кодировка контента HTTP-ответа
}

// NewResponseData создает новый объект ResponseData с пустыми значениями.
func NewResponseData() *ResponseData {
	return &ResponseData{}
}

// LoggingResponseWriter расширяет ResponseWriter и используется для записи
// данных об ответе, таких как размер, тип контента и кодировка.
type LoggingResponseWriter struct {
	http.ResponseWriter
	rd *ResponseData
}

// NewLoggingResponseWriter создает новый объект LoggingResponseWriter.
// Он оборачивает стандартный ResponseWriter и связывает его с объектом ResponseData.
func NewLoggingResponseWriter(rw http.ResponseWriter, rd *ResponseData) *LoggingResponseWriter {
	return &LoggingResponseWriter{ResponseWriter: rw, rd: rd}
}

// Write записывает данные в ResponseWriter и обновляет информацию о размере ответа
// и кодировке контента.
func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	// Записываем данные в ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	// Обновляем размер ответа
	r.rd.respSizeB += size
	// Обновляем кодировку контента, взяв из заголовков
	r.rd.respContentEncoding = r.Header().Get("Content-Encoding")
	return size, err
}

// WriteHeader записывает статус код в ResponseWriter и обновляет информацию
// о статусе ответа и типе контента.
func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	// Записываем статус код в ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	// Обновляем тип контента из заголовков ответа
	r.rd.respContentType = r.ResponseWriter.Header().Get("Content-Type")
	// Обновляем статус код
	r.rd.statusCode = statusCode
}
