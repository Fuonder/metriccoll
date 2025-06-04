package interceptors

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	pb "github.com/Fuonder/metriccoll.git/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
)

func gzipData(data []byte) ([]byte, error) {
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

func gunzipData(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	gz, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return io.ReadAll(gz)
}

func UnaryGzipRequestInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Log.Info("Unzipping Client request")
		md, _ := metadata.FromIncomingContext(ctx)
		enc := md.Get("content-encoding")
		if len(enc) == 0 || enc[0] != "gzip" {
			return handler(ctx, req)
		}
		encryptedMsg, ok := req.(*pb.EncryptedMessage)
		if !ok {
			return nil, status.Error(codes.Internal, "expected EncryptedMessage request")
		}

		if len(encryptedMsg.Blob) == 0 {
			return nil, status.Error(codes.InvalidArgument, "blob is empty")
		}
		unzipped, err := gunzipData(encryptedMsg.Blob)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "gunzip failed: %v", err)
		}
		encryptedMsg.Blob = unzipped
		logger.Log.Info("Unzipping Client request", zap.String("status", "Success"))
		return handler(ctx, encryptedMsg)
	}
}

func UnaryGzipResponseInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, err
		}
		logger.Log.Info("Zipping Server Response")
		md, _ := metadata.FromIncomingContext(ctx)
		accept := md.Get("content-encoding")
		if len(accept) == 0 || accept[0] != "gzip" {
			return resp, nil
		}

		switch r := resp.(type) {
		case *pb.UpdateMetricsResponse:
			logger.Log.Info("Got Response of Type *pb.UpdateMetricsResponse")
			r.Blob, err = gzipData(r.Blob)
		case *pb.UpdateMetricResponse:
			logger.Log.Info("Got Response of Type *pb.UpdateMetricResponse")
			r.Blob, err = gzipData(r.Blob)
		case *pb.ListMetricsResponse:
			logger.Log.Info("Got Response of Type *pb.ListMetricsResponse")
			r.Blob, err = gzipData(r.Blob)
		case *pb.GetMetricResponse:
			logger.Log.Info("Got Response of Type *pb.GetMetricResponse")
			r.Value, err = gzipData(r.Value)
		default:
			logger.Log.Info("Got unknown Response Type")
			return nil, status.Errorf(codes.Internal, "unknown Response Type")
		}
		if err != nil {
			return nil, status.Errorf(codes.Internal, "gzip failed: %v", err)
		}

		grpc.SendHeader(ctx, metadata.Pairs("content-encoding", "gzip"))
		return resp, nil
	}
}
