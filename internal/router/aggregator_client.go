package router

import (
	"context"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AggregatorClient struct {
	address string
	markets []string
	store   *BookStore
	logger  zerolog.Logger
}

func NewAggregatorClient(
	address string,
	markets []string,
	store *BookStore,
	logger zerolog.Logger,
) *AggregatorClient {
	return &AggregatorClient{
		address: address,
		markets: markets,
		store:   store,
		logger:  logger,
	}
}

func (a *AggregatorClient) Run(ctx context.Context) error {
	reconnectDelay := 2 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := a.runOnce(ctx)
		if err != nil && ctx.Err() == nil {
			a.logger.Error().
				Err(err).
				Str("address", a.address).
				Dur("reconnect_delay", reconnectDelay).
				Msg("aggregator stream disconnected; reconnecting")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnectDelay):
		}
	}
}

func (a *AggregatorClient) runOnce(ctx context.Context) error {
	conn, err := grpc.DialContext(
		ctx,
		a.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewAggregatorClient(conn)

	a.logger.Info().
		Str("address", a.address).
		Msg("connected to aggregator")

	for _, market := range a.markets {
		marketID := market

		go func() {
			if err := a.streamMarket(ctx, client, marketID); err != nil && ctx.Err() == nil {
				a.logger.Error().
					Err(err).
					Str("market", marketID).
					Msg("market stream stopped")
			}
		}()
	}

	<-ctx.Done()
	return ctx.Err()
}

func (a *AggregatorClient) streamMarket(
	ctx context.Context,
	client pb.AggregatorClient,
	market string,
) error {
	stream, err := client.StreamConsolidatedBook(ctx, &pb.ConsolidatedBookRequest{
		CanonicalMarketId: market,
	})
	if err != nil {
		return err
	}

	for {
		book, err := stream.Recv()
		if err != nil {
			return err
		}

		a.store.Upsert(book)
	}
}
