package aggregator

import (
	"context"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
)

type Server struct {
	pb.UnimplementedAggregatorServer

	nodeID string
	store  *Store
}

func NewServer(nodeID string, store *Store) *Server {
	return &Server{
		nodeID: nodeID,
		store:  store,
	}
}

func (s *Server) GetConsolidatedBook(
	ctx context.Context,
	req *pb.ConsolidatedBookRequest,
) (*pb.ConsolidatedBook, error) {
	return s.store.GetConsolidatedBook(req.CanonicalMarketId), nil
}

func (s *Server) StreamConsolidatedBook(
	req *pb.ConsolidatedBookRequest,
	stream pb.Aggregator_StreamConsolidatedBookServer,
) error {
	ch := s.store.Subscribe()
	defer s.store.Unsubscribe(ch)

	initial := s.store.GetConsolidatedBook(req.CanonicalMarketId)
	if err := stream.Send(initial); err != nil {
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case book := <-ch:
			if req.CanonicalMarketId != "" &&
				book.CanonicalMarketId != req.CanonicalMarketId {
				continue
			}

			if err := stream.Send(book); err != nil {
				return err
			}
		}
	}
}

func (s *Server) AggregatorHealth(
	ctx context.Context,
	req *pb.HealthRequest,
) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		NodeId:           s.nodeID,
		Venue:            "AGGREGATOR",
		Alive:            true,
		Stale:            false,
		LastMessageAgeMs: 0,
		LastSequence:     0,
		Status:           "OK",
	}, nil
}

func nowNs() int64 {
	return time.Now().UnixNano()
}
