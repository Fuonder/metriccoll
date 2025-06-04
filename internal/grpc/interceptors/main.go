package interceptors

import (
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"google.golang.org/grpc"
)

// NewUnaryInterceptorChain возвращает упорядоченную цепочку перехватчиков gRPC.
func NewUnaryInterceptorChain(hmacKey string, cipher certmanager.TLSDecipher, trustedSubnet string) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		UnaryTrustedSubnetInterceptor(trustedSubnet), // check subnet
		UnaryServerHMACInterceptor(hmacKey),          // validate HMAC
		UnaryDecryptionInterceptor(cipher),           // Decrypt message
		UnaryGzipRequestInterceptor(),                // Unzip message
		UnaryGzipResponseInterceptor(),               // invoke handler and compress if needed
		UnaryServerResponseHMACInterceptor(hmacKey),  // write HMAC to response
	)
}
