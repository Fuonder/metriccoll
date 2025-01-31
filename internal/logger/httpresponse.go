package logger

import "net/http"

type ResponseData struct {
	statusCode          int
	respSizeB           int
	respContentType     string
	respContentEncoding string
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
	r.rd.respContentEncoding = r.Header().Get("Content-Encoding")
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.rd.respContentType = r.ResponseWriter.Header().Get("Content-Type")
	r.rd.statusCode = statusCode
}
