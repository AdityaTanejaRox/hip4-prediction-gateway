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
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/venue/polymarket"

	"google.golang.org/grpc"
)

func main() {
	cfgPath := flag.String("config", "configs/polymarket.yaml", "path to config file")
	flag.Parse()

	logger := logx.New()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load polymarket config")
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	localBook := book.NewBook(
		domain.VenuePolyMarket,
		cfg.Polymarket.VenueMarketID,
		cfg.Polymarket.CanonicalMarketID,
		time.Duration(cfg.Polymarket.StaleAfterMS)*time.Millisecond,
	)

	var updates <-chan domain.TopOfBook

	if cfg.MockMode {
		mockFeed := polymarket.NewMockFeed(localBook)
		updates = mockFeed.Updates()

		go func() {
			if err := mockFeed.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Error().Err(err).Msg("polymarket mock feed stopped")
			}
		}()

		logger.Info().Msg("polymarket-node running in mock mode")
	} else {
		wsFeed := polymarket.NewWSFeed(
			polymarket.WSFeedConfig{
				WebSocketURL:      cfg.Polymarket.WebSocketURL,
				AssetIDs:          cfg.Polymarket.AssetIDs,
				VenueMarketID:     cfg.Polymarket.VenueMarketID,
				CanonicalMarketID: cfg.Polymarket.CanonicalMarketID,
				ReconnectDelay:    2 * time.Second,
			},
			localBook,
			logger,
		)

		updates = wsFeed.Updates()

		go func() {
			if err := wsFeed.Run(ctx); err != nil && ctx.Err() == nil {
				logger.Error().Err(err).Msg("polymarket websocket feed stopped")
			}
		}()

		logger.Info().
			Str("url", cfg.Polymarket.WebSocketURL).
			Strs("asset_ids", cfg.Polymarket.AssetIDs).
			Msg("polymarket-node running in real websocket mode")
	}

	listener, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to listen")
	}

	grpcServer := grpc.NewServer()

	server := polymarket.NewServer(
		cfg.NodeID,
		domain.VenuePolyMarket,
		localBook,
		updates,
	)

	pb.RegisterMarketDataNodeServer(grpcServer, server)

	logger.Info().
		Str("addr", cfg.GRPCListenAddr).
		Bool("mock_mode", cfg.MockMode).
		Msg("polymarket-node started")

	go func() {
		<-ctx.Done()
		logger.Info().Msg("shutting down polymarket-node")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal().Err(err).Msg("polymarket grpc server stopped")
	}
}
