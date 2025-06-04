package provider

import (
	"context"
	"encoding/json"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	pb "github.com/Fuonder/metriccoll.git/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCProvider struct {
	pb.UnimplementedMetricsServer
	mReader      storage.MetricReader
	mWriter      storage.MetricWriter
	mFileHandler storage.MetricFileHandler
	mDBHandler   storage.MetricDatabaseHandler
}

func NewGRPCProvider(
	mReader storage.MetricReader,
	mWriter storage.MetricWriter,
	mFileHandler storage.MetricFileHandler,
	mDBHandler storage.MetricDatabaseHandler,
) *GRPCProvider {
	return &GRPCProvider{
		mReader:      mReader,
		mWriter:      mWriter,
		mFileHandler: mFileHandler,
		mDBHandler:   mDBHandler,
	}
}

func (p *GRPCProvider) UpdateMetric(ctx context.Context, in *pb.EncryptedMessage) (*pb.UpdateMetricResponse, error) {
	if in == nil || len(in.Blob) == 0 {
		return &pb.UpdateMetricResponse{Error: "empty blob"}, status.Error(codes.InvalidArgument, "empty blob")
	}

	var metric models.Metrics
	if err := json.Unmarshal(in.Blob, &metric); err != nil {
		return &pb.UpdateMetricResponse{Error: "invalid JSON"}, status.Errorf(codes.InvalidArgument, "invalid JSON: %v", err)
	}

	if err := p.mWriter.AppendMetric(metric); err != nil {
		return &pb.UpdateMetricResponse{Error: "append failed"}, status.Errorf(codes.Internal, "append error: %v", err)
	}

	updated, err := p.mReader.GetMetricByName(metric.ID, metric.MType)
	if err != nil {
		return &pb.UpdateMetricResponse{Error: "fetch failed"}, status.Errorf(codes.Internal, "read error: %v", err)
	}

	respBytes, err := json.Marshal(updated)
	if err != nil {
		return &pb.UpdateMetricResponse{Error: "marshal failed"}, status.Errorf(codes.Internal, "marshal error: %v", err)
	}

	return &pb.UpdateMetricResponse{Blob: respBytes}, nil
}

func (p *GRPCProvider) UpdateMetrics(ctx context.Context, in *pb.EncryptedMessage) (*pb.UpdateMetricsResponse, error) {
	if in == nil || len(in.Blob) == 0 {
		return &pb.UpdateMetricsResponse{Error: "empty blob"}, status.Error(codes.InvalidArgument, "empty blob")
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(in.Blob, &metrics); err != nil {
		return &pb.UpdateMetricsResponse{Error: "invalid JSON array"}, status.Errorf(codes.InvalidArgument, "invalid JSON: %v", err)
	}

	if err := p.mWriter.AppendMetrics(metrics); err != nil {
		return &pb.UpdateMetricsResponse{Error: "append error"}, status.Errorf(codes.Internal, "append failed: %v", err)
	}

	updatedMetrics := make([]models.Metrics, 0, len(metrics))
	for _, m := range metrics {
		mt, err := p.mReader.GetMetricByName(m.ID, m.MType)
		if err != nil {
			return &pb.UpdateMetricsResponse{Error: "fetch failed"}, status.Errorf(codes.Internal, "fetch error: %v", err)
		}
		updatedMetrics = append(updatedMetrics, mt)
	}

	respBlob, err := json.Marshal(updatedMetrics)
	if err != nil {
		return &pb.UpdateMetricsResponse{Error: "marshal failed"}, status.Errorf(codes.Internal, "marshal error: %v", err)
	}

	return &pb.UpdateMetricsResponse{Blob: respBlob}, nil
}

func (p *GRPCProvider) GetMetric(ctx context.Context, in *pb.EncryptedMessage) (*pb.GetMetricResponse, error) {
	if in == nil || len(in.Blob) == 0 {
		return &pb.GetMetricResponse{Error: "empty blob"}, status.Error(codes.InvalidArgument, "empty blob")
	}

	var query models.Metrics
	if err := json.Unmarshal(in.Blob, &query); err != nil {
		return &pb.GetMetricResponse{Error: "invalid JSON"}, status.Errorf(codes.InvalidArgument, "invalid JSON: %v", err)
	}

	result, err := p.mReader.GetMetricByName(query.ID, query.MType)
	if err != nil {
		return &pb.GetMetricResponse{Error: "metric not found"}, status.Errorf(codes.NotFound, "metric not found: %v", err)
	}

	respBlob, err := json.Marshal(result)
	if err != nil {
		return &pb.GetMetricResponse{Error: "marshal failed"}, status.Errorf(codes.Internal, "marshal error: %v", err)
	}

	return &pb.GetMetricResponse{Value: respBlob}, nil
}

func (p *GRPCProvider) ListMetrics(ctx context.Context, _ *pb.EncryptedMessage) (*pb.ListMetricsResponse, error) {
	metrics := p.mReader.GetAllMetrics()
	respBlob, err := json.Marshal(metrics)
	if err != nil {
		return &pb.ListMetricsResponse{Error: "marshal failed"}, status.Errorf(codes.Internal, "marshal error: %v", err)
	}
	return &pb.ListMetricsResponse{Blob: respBlob}, nil
}
