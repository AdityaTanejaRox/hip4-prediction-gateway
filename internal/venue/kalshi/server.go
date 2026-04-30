package kalshi

import (
	"context"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

type Server struct {
	pb.UnimplementedMarketDataNodeServer

	nodeID string
	venue  domain.Venue

	book    *book.Book
	updates <-chan domain.TopOfBook
}

func NewServer(
	nodeID string,
	venue domain.Venue,
	book *book.Book,
	updates <-chan domain.TopOfBook,
) *Server {
	return &Server{
		nodeID:  nodeID,
		venue:   venue,
		book:    book,
		updates: updates,
	}
}

func (s *Server) StreamTopOfBook(
	req *pb.StreamRequest,
	stream pb.MarketDataNode_StreamTopOfBookServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case update := <-s.updates:
			msg := &pb.TopOfBookUpdate{
				Venue:             string(update.Venue),
				VenueMarketId:     update.VenueMarketID,
				CanonicalMarketId: update.CanonicalMarketID,

				YesBidPriceBps: int64(update.YesBidPriceBps),
				YesBidQty:      update.YesBidQty,

				YesAskPriceBps: int64(update.YesAskPriceBps),
				YesAskQty:      update.YesAskQty,

				ExchangeTsNs: update.ExchangeTs.UnixNano(),
				ReceiveTsNs:  update.ReceiveTs.UnixNano(),

				Sequence: update.Sequence,
				Stale:    update.Stale,
			}

			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
}

func (s *Server) GetSnapshot(
	ctx context.Context,
	req *pb.SnapshotRequest,
) (*pb.BookSnapshot, error) {
	bids, asks, sequence, stale := s.book.Snapshot(10)

	return &pb.BookSnapshot{
		Venue:             string(s.venue),
		VenueMarketId:     "",
		CanonicalMarketId: req.CanonicalMarketId,
		YesBids:           toProtoLevels(bids),
		YesAsks:           toProtoLevels(asks),
		Sequence:          sequence,
		ReceiveTsNs:       time.Now().UnixNano(),
		Stale:             stale,
	}, nil
}

func (s *Server) Health(
	ctx context.Context,
	req *pb.HealthRequest,
) (*pb.HealthResponse, error) {
	_, _, sequence, stale := s.book.Snapshot(1)

	status := "OK"
	if stale {
		status = "STALE"
	}

	return &pb.HealthResponse{
		NodeId:           s.nodeID,
		Venue:            string(s.venue),
		Alive:            true,
		Stale:            stale,
		LastMessageAgeMs: 0,
		LastSequence:     sequence,
		Status:           status,
	}, nil
}

func toProtoLevels(levels []domain.PriceLevel) []*pb.PriceLevel {
	out := make([]*pb.PriceLevel, 0, len(levels))

	for _, level := range levels {
		out = append(out, &pb.PriceLevel{
			PriceBps: int64(level.PriceBps),
			Quantity: level.Quantity,
		})
	}

	return out
}
