package interceptors

import (
	"context"
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	pb "github.com/Fuonder/metriccoll.git/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryDecryptionInterceptor(cipher certmanager.TLSDecipher) grpc.UnaryServerInterceptor {
	logger.Log.Info("TRYING TO DECIPHER BODY")
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		encryptedMsg, ok := req.(*pb.EncryptedMessage)
		if !ok {
			return nil, status.Error(codes.Internal, "expected EncryptedMessage request")
		}
		if len(encryptedMsg.Blob) == 0 {
			return nil, status.Error(codes.InvalidArgument, "blob is empty")
		}
		decryptedBlob, err := cipher.Decrypt(encryptedMsg.Blob)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "decryption failed: %v", err)
		}

		encryptedMsg.Blob = decryptedBlob
		logger.Log.Info("Blob decrypted successfully")

		return handler(ctx, encryptedMsg)
	}
}
