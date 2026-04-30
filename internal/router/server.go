package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/arbitrage"
)

type Server struct {
	pb.UnimplementedRouterServer

	store    *BookStore
	selector *Selector
	scanner  *arbitrage.Scanner

	mu          sync.Mutex
	subscribers map[chan *pb.ArbitrageOpportunity]struct{}

	lastPublished map[string]time.Time
	dedupeWindow  time.Duration
}

func NewServer(
	store *BookStore,
	selector *Selector,
	scanner *arbitrage.Scanner,
) *Server {
	return &Server{
		store:         store,
		selector:      selector,
		scanner:       scanner,
		subscribers:   make(map[chan *pb.ArbitrageOpportunity]struct{}),
		lastPublished: make(map[string]time.Time),
		dedupeWindow:  2 * time.Second,
	}
}

func (s *Server) SubmitIntent(
	ctx context.Context,
	req *pb.OrderIntent,
) (*pb.RouteDecision, error) {
	book, ok := s.store.Get(req.CanonicalMarketId)
	if !ok {
		return nil, fmt.Errorf("no book available for market: %s", req.CanonicalMarketId)
	}

	return s.selector.SelectRoute(book, req.Side, req.Quantity)
}

func (s *Server) StreamOpportunities(
	req *pb.OpportunityRequest,
	stream pb.Router_StreamOpportunitiesServer,
) error {
	ch := make(chan *pb.ArbitrageOpportunity, 1024)

	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		close(ch)
		s.mu.Unlock()
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case opportunity := <-ch:
			if req.CanonicalMarketId != "" &&
				opportunity.CanonicalMarketId != req.CanonicalMarketId {
				continue
			}

			if err := stream.Send(opportunity); err != nil {
				return err
			}
		}
	}
}

func (s *Server) PublishOpportunities(book *pb.ConsolidatedBook) {
	opportunities := s.scanner.Scan(book)
	if len(opportunities) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, opportunity := range opportunities {
		key := opportunityKey(opportunity)

		if last, ok := s.lastPublished[key]; ok {
			if now.Sub(last) < s.dedupeWindow {
				continue
			}
		}

		s.lastPublished[key] = now

		for ch := range s.subscribers {
			select {
			case ch <- opportunity:
			default:
			}
		}
	}
}

func opportunityKey(opp *pb.ArbitrageOpportunity) string {
	return fmt.Sprintf(
		"%s|%s|%s|%d|%d",
		opp.CanonicalMarketId,
		opp.BuyVenue,
		opp.SellVenue,
		opp.BuyPriceBps,
		opp.SellPriceBps,
	)
}
