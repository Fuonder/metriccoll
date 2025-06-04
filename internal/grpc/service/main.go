package service

import (
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"github.com/Fuonder/metriccoll.git/internal/grpc/interceptors"
	"github.com/Fuonder/metriccoll.git/internal/grpc/provider"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	pb "github.com/Fuonder/metriccoll.git/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
)

type Service struct {
	provider      *provider.GRPCProvider
	ipaddr        string
	hashKey       string
	cipherManager certmanager.TLSDecipher
	trustedSubnet string
	grpcServer    *grpc.Server
	listener      net.Listener
}

func NewService(
	mReader storage.MetricReader,
	mWriter storage.MetricWriter,
	mFileHandler storage.MetricFileHandler,
	mDBHandler storage.MetricDatabaseHandler,
	cipherManager certmanager.TLSDecipher,
	trustedSubnet string,
	ipaddr string,
	hashKey string) *Service {

	p := provider.NewGRPCProvider(mReader,
		mWriter,
		mFileHandler,
		mDBHandler)

	s := Service{
		provider:      p,
		ipaddr:        ipaddr,
		cipherManager: cipherManager,
		trustedSubnet: trustedSubnet,
		hashKey:       hashKey}
	return &s
}

func (s *Service) Run() error {
	logger.Log.Info("Binding to ", zap.Any("ip", s.ipaddr))
	listen, err := net.Listen("tcp", s.ipaddr)
	if err != nil {
		return err
	}
	s.listener = listen

	srv := grpc.NewServer(
		interceptors.NewUnaryInterceptorChain(s.hashKey, s.cipherManager, s.trustedSubnet))
	s.grpcServer = srv

	pb.RegisterMetricsServer(srv, s.provider)

	return srv.Serve(listen)
}

func (s *Service) Close() error {
	if s.grpcServer != nil {
		logger.Log.Info("Shutting down gRPC server")
		s.grpcServer.GracefulStop()
	}
	if s.listener != nil {
		_ = s.listener.Close() // optional: close listener if still open
	}
	return nil
}
