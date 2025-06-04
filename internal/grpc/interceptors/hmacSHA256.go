package interceptors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	pb "github.com/Fuonder/metriccoll.git/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func calculateHMAC(message []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(message)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func verifyHMAC(message []byte, receivedMAC, key string) bool {
	expectedMAC := calculateHMAC(message, key)
	return hmac.Equal([]byte(receivedMAC), []byte(expectedMAC))
}

func UnaryServerHMACInterceptor(hmacKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Log.Info("Checking HMAC from HashSHA256")
		if hmacKey == "" {
			logger.Log.Info("HMAC KEY NOT SET")
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		signatures := md.Get("HashSHA256")
		if len(signatures) == 0 {
			return handler(ctx, req)
		}
		msg, ok := req.(*pb.EncryptedMessage)
		if !ok {
			return nil, status.Error(codes.Internal, "request is not pb.EncryptedMessage")
		}
		if !verifyHMAC(msg.Blob, signatures[0], hmacKey) {
			return nil, status.Error(codes.InvalidArgument, "invalid HMAC")
		}
		logger.Log.Info("HMAC - OK")
		return handler(ctx, req)
	}
}

func UnaryServerResponseHMACInterceptor(hmacKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil || hmacKey == "" {
			return resp, err
		}
		msg, ok := resp.(proto.Message)
		if !ok {
			return resp, nil
		}
		raw, _ := proto.Marshal(msg)
		signature := calculateHMAC(raw, hmacKey)
		grpc.SetHeader(ctx, metadata.Pairs("HashSHA256", signature))
		return resp, nil
	}
}
