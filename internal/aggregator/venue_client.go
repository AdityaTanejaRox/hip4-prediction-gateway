package aggregator

import (
	"context"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/config"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type VenueStreamClient struct {
	nodeConfig config.VenueNodeConfig
	markets    []string
	store      *Store
	logger     zerolog.Logger
}

func NewVenueStreamClient(
	nodeConfig config.VenueNodeConfig,
	markets []string,
	store *Store,
	logger zerolog.Logger,
) *VenueStreamClient {
	return &VenueStreamClient{
		nodeConfig: nodeConfig,
		markets:    markets,
		store:      store,
		logger:     logger,
	}
}

func (v *VenueStreamClient) Run(ctx context.Context) error {
	reconnectDelay := 2 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := v.runOnce(ctx)
		if err != nil && ctx.Err() == nil {
			v.logger.Error().
				Err(err).
				Str("venue", v.nodeConfig.Venue).
				Str("address", v.nodeConfig.Address).
				Dur("reconnect_delay", reconnectDelay).
				Msg("venue stream disconnected; reconnecting")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnectDelay):
		}
	}
}

func (v *VenueStreamClient) runOnce(ctx context.Context) error {
	conn, err := grpc.DialContext(
		ctx,
		v.nodeConfig.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewMarketDataNodeClient(conn)

	stream, err := client.StreamTopOfBook(ctx, &pb.StreamRequest{
		CanonicalMarketIds: v.markets,
	})
	if err != nil {
		return err
	}

	v.logger.Info().
		Str("venue", v.nodeConfig.Venue).
		Str("address", v.nodeConfig.Address).
		Msg("connected to venue stream")

	for {
		update, err := stream.Recv()
		if err != nil {
			return err
		}

		v.store.Upsert(VenueTopOfBook{
			Venue:             update.Venue,
			VenueMarketID:     update.VenueMarketId,
			CanonicalMarketID: update.CanonicalMarketId,

			YesBidPriceBps: domain.PriceBps(update.YesBidPriceBps),
			YesBidQty:      update.YesBidQty,

			YesAskPriceBps: domain.PriceBps(update.YesAskPriceBps),
			YesAskQty:      update.YesAskQty,

			ReceiveTs: time.Unix(0, update.ReceiveTsNs),
			Sequence:  update.Sequence,
			Stale:     update.Stale,
		})
	}
}
