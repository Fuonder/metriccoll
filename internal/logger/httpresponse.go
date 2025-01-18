package logger

import "net/http"

type ResponseData struct {
	statusCode int
	respSizeB  int
}

func NewResponseData() *ResponseData {
	return &ResponseData{}
}

type LoggingResponseWriter struct {
	http.ResponseWriter
	rd *ResponseData
}

func NewLoggingResponseWriter(rw http.ResponseWriter, rd *ResponseData) *LoggingResponseWriter {
	return &LoggingResponseWriter{ResponseWriter: rw, rd: rd}
}

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.rd.respSizeB += size
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.rd.statusCode = statusCode
}
