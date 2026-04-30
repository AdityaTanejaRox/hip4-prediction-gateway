package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/config"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/logx"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/venue/hip4"

	"google.golang.org/grpc"
)

func main() {
	cfgPath := flag.String("config", "configs/hip4-testnet.yaml", "path to config file")
	flag.Parse()

	logger := logx.New()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	localBook := book.NewBook(
		domain.VenueHIP4,
		cfg.Hyperliquid.VenueMarketID,
		cfg.Hyperliquid.CanonicalMarketID,
		time.Duration(cfg.Hyperliquid.StaleAfterMS)*time.Millisecond,
	)

	mockFeed := hip4.NewMockFeed(localBook)

	go func() {
		if err := mockFeed.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error().Err(err).Msg("mock feed stopped")
		}
	}()

	listener, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen")
	}

	grpcServer := grpc.NewServer()

	hip4Server := hip4.NewServer(
		cfg.NodeID,
		domain.VenueHIP4,
		localBook,
		mockFeed.Updates(),
	)

	pb.RegisterMarketDataNodeServer(grpcServer, hip4Server)

	logger.Info().
		Str("addr", cfg.GRPCListenAddr).
		Bool("mock_mode", cfg.MockMode).
		Msg("hip4-node started")

	go func() {
		<-ctx.Done()
		logger.Info().Msg("Shutting down hip4-node")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("GRPC server stopped")
	}
}
