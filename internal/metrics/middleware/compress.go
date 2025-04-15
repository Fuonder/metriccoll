package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"github.com/Fuonder/metriccoll.git/internal/logger"
	"go.uber.org/zap"
)

func GzipCompress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %v", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	logger.Log.Info("Compression stats",
		zap.Int("Given", len(data)),
		zap.Int("Compressed", len(buffer.Bytes())))
	return buffer.Bytes(), nil
}

func gzipDecompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed init compress reader: %v", err)
	}
	defer reader.Close()

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed decompress data: %v", err)
	}
	return buffer.Bytes(), nil
}
