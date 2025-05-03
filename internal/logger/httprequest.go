package logger

import (
	"fmt"
	"time"
)

// RequestData содержит информацию о HTTP-запросе.
type RequestData struct {
	url       string    // URL запроса
	method    string    // HTTP метод запроса (например, GET, POST)
	timeStart time.Time // Время получения запроса
}

// NewRequestData создает новый объект RequestData с пустыми значениями
// для URL и метода, и устанавливает текущее время как время получения запроса.
func NewRequestData() *RequestData {
	return &RequestData{url: "", method: "", timeStart: time.Now()}
}

// Set устанавливает URL и метод для объекта RequestData.
func (r *RequestData) Set(uri string, method string) {
	r.url = uri
	r.method = method
}

// Deprecated: GetTimeSpent не используется в новых версиях.
// Используйте напрямую time.Since для поля RequestData.timeStart.
// GetTimeSpent возвращает продолжительность времени, прошедшего с момента получения запроса.
// Если время получения запроса не задано, возвращает ошибку.
func (r *RequestData) GetTimeSpent() (time.Duration, error) {
	if r.timeStart.IsZero() {
		return time.Duration(0), fmt.Errorf("request start time must be greater than zero")
	}
	return time.Since(r.timeStart), nil

}

// GetMethod возвращает HTTP метод запроса.
func (r *RequestData) GetMethod() string {
	return r.method
}

// GetURI возвращает URL запроса.
func (r *RequestData) GetURI() string {
	return r.method
}

// String возвращает строковое представление объекта RequestData,
// включая URL, метод и время получения запроса.
func (r *RequestData) String() string {
	return fmt.Sprintf("request:\n\tURI:\t%s\n\tMETHOD:\t%s\n\tTIMESTART:\t%s\n\n",
		r.url, r.method, r.timeStart.String())
}
