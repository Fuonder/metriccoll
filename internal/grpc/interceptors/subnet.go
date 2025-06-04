package interceptors

import (
	"context"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"strings"
)

func isIPTrusted(ipStr, cidr string) bool {
	ip := net.ParseIP(ipStr)
	_, subnet, err := net.ParseCIDR(cidr)
	return err == nil && ip != nil && subnet.Contains(ip)
}

func UnaryTrustedSubnetInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Log.Info("Checking subnet form X-Real-IP")
		if trustedSubnet == "" {
			logger.Log.Info("Checking subnet NOT NEEDED")
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		realIPs := md.Get("X-Real-IP")
		if len(realIPs) == 0 || !isIPTrusted(strings.TrimSpace(realIPs[0]), trustedSubnet) {
			return nil, status.Error(codes.PermissionDenied, "IP not in trusted subnet")
		}
		logger.Log.Info("IP TRUSTED")
		return handler(ctx, req)
	}
}
