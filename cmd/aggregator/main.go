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
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/aggregator"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/config"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/logx"
	"google.golang.org/grpc"
)

func main() {
	cfgPath := flag.String("config", "configs/aggregator.yaml", "path to config file")
	flag.Parse()

	logger := logx.New()

	cfg, err := config.LoadAggregator(*cfgPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load aggregator config")
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	store := aggregator.NewStore(
		time.Duration(cfg.QuoteStaleAfterMS) * time.Millisecond,
	)

	for _, venueNode := range cfg.VenueNodes {
		client := aggregator.NewVenueStreamClient(
			venueNode,
			cfg.Markets,
			store,
			logger,
		)

		go func() {
			if err := client.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Error().Err(err).Msg("venue client stopped")
			}
		}()
	}

	listener, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen")
	}

	grpcServer := grpc.NewServer()

	server := aggregator.NewServer(cfg.NodeID, store)
	pb.RegisterAggregatorServer(grpcServer, server)

	logger.Info().
		Str("addr", cfg.GRPCListenAddr).
		Msg("aggregator started")

	go func() {
		<-ctx.Done()
		logger.Info().Msg("shutting down aggregator")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("aggregator grpc server stopped")
	}
}
