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
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/arbitrage"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/config"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/logx"
	routerpkg "github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/router"
	"google.golang.org/grpc"
)

func main() {
	cfgPath := flag.String("config", "configs/router.yaml", "path to router config")
	flag.Parse()

	logger := logx.New()

	cfg, err := config.LoadRouter(*cfgPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load router config")
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	bookStore := routerpkg.NewBookStore()
	selector := routerpkg.NewSelector()

	scanner := arbitrage.NewScanner(arbitrage.ScannerConfig{
		MinNetEdgeBps:      cfg.Routing.MinNetEdgeBps,
		DefaultFeeBps:      cfg.Routing.DefaultFeeBps,
		DefaultSlippageBps: cfg.Routing.DefaultSlippageBps,
	})

	aggregatorClient := routerpkg.NewAggregatorClient(
		cfg.Aggregator.Address,
		cfg.Markets,
		bookStore,
		logger,
	)

	go func() {
		if err := aggregatorClient.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error().Err(err).Msg("aggregator client stopped")
		}
	}()

	routerServer := routerpkg.NewServer(
		bookStore,
		selector,
		scanner,
	)

	go routerpkg.RunOpportunityLoop(
		ctx,
		bookStore,
		routerServer,
		cfg.Markets,
		250*time.Millisecond,
	)

	listener, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen")
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRouterServer(grpcServer, routerServer)

	logger.Info().
		Str("addr", cfg.GRPCListenAddr).
		Str("aggregator", cfg.Aggregator.Address).
		Msg("router started")

	go func() {
		<-ctx.Done()
		logger.Info().Msg("shutting down router")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("router grpc server stopped")
	}
}
