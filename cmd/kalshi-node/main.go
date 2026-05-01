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
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/venue/kalshi"

	"google.golang.org/grpc"
)

func main() {
	cfgPath := flag.String("config", "configs/kalshi.yaml", "path to config file")
	flag.Parse()

	logger := logx.New()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load kalshi config")
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	localBook := book.NewBook(
		domain.VenueKalshi,
		cfg.Kalshi.VenueMarketID,
		cfg.Kalshi.CanonicalMarketID,
		time.Duration(cfg.Kalshi.StaleAfterMS)*time.Millisecond,
	)

	var updates <-chan domain.TopOfBook

	if cfg.MockMode {
		mockFeed := kalshi.NewMockFeed(localBook)
		updates = mockFeed.Updates()

		go func() {
			if err := mockFeed.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Error().Err(err).Msg("kalshi mock feed stopped")
			}
		}()

		logger.Info().Msg("kalshi-node running in mock mode")
	} else {
		wsFeed := kalshi.NewWSFeed(
			kalshi.WSFeedConfig{
				WebSocketURL:      cfg.Kalshi.WebSocketURL,
				VenueMarketID:     cfg.Kalshi.VenueMarketID,
				CanonicalMarketID: cfg.Kalshi.CanonicalMarketID,
				APIKeyEnv:         cfg.Kalshi.ApiKeyEnv,
				APISecretEnv:      cfg.Kalshi.ApiSecretEnv,
				ReconnectDelay:    2 * time.Second,
			},
			localBook,
			logger,
		)

		updates = wsFeed.Updates()

		go func() {
			if err := wsFeed.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Error().Err(err).Msg("kalshi websocket feed stopped")
			}
		}()

		logger.Info().
			Str("url", cfg.Kalshi.WebSocketURL).
			Str("market", cfg.Kalshi.VenueMarketID).
			Msg("kalshi-node running in real websocket mode")
	}

	listener, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen")
	}

	grpcServer := grpc.NewServer()

	server := kalshi.NewServer(
		cfg.NodeID,
		domain.VenueKalshi,
		localBook,
		updates,
	)

	pb.RegisterMarketDataNodeServer(grpcServer, server)

	logger.Info().
		Str("addr", cfg.GRPCListenAddr).
		Bool("mock_mode", cfg.MockMode).
		Msg("kalshi-node started")

	go func() {
		<-ctx.Done()
		logger.Info().Msg("shutting down kalshi-node")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("kalshi grpc server stopped")
	}
}
