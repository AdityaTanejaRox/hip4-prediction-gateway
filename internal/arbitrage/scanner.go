package arbitrage

import (
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
)

type ScannerConfig struct {
	MinNetEdgeBps      int64
	DefaultFeeBps      int64
	DefaultSlippageBps int64
}

type Scanner struct {
	cfg ScannerConfig
}

func NewScanner(cfg ScannerConfig) *Scanner {
	return &Scanner{
		cfg: cfg,
	}
}

func (s *Scanner) Scan(book *pb.ConsolidatedBook) []*pb.ArbitrageOpportunity {
	if book == nil {
		return nil
	}

	bestAsk := firstHealthyAsk(book)
	bestBid := firstHealthyBid(book)

	if bestAsk == nil || bestBid == nil {
		return nil
	}

	if bestAsk.Venue == bestBid.Venue {
		return nil
	}

	grossEdgeBps := bestBid.PriceBps - bestAsk.PriceBps
	if grossEdgeBps <= 0 {
		return nil
	}

	totalCostBps := s.cfg.DefaultFeeBps*2 + s.cfg.DefaultSlippageBps
	netEdgeBps := grossEdgeBps - totalCostBps

	if netEdgeBps < s.cfg.MinNetEdgeBps {
		return nil
	}

	return []*pb.ArbitrageOpportunity{
		{
			CanonicalMarketId: book.CanonicalMarketId,
			BuyVenue:          bestAsk.Venue,
			SellVenue:         bestBid.Venue,
			BuyPriceBps:       bestAsk.PriceBps,
			SellPriceBps:      bestBid.PriceBps,
			GrossEdgeBps:      grossEdgeBps,
			NetEdgeBps:        netEdgeBps,
			DetectedTsNs:      time.Now().UnixNano(),
		},
	}
}

func firstHealthyAsk(book *pb.ConsolidatedBook) *pb.VenueQuote {
	for _, ask := range book.YesAsks {
		if ask.Stale {
			continue
		}

		if ask.Quantity <= 0 {
			continue
		}

		return ask
	}

	return nil
}

func firstHealthyBid(book *pb.ConsolidatedBook) *pb.VenueQuote {
	for _, bid := range book.YesBids {
		if bid.Stale {
			continue
		}

		if bid.Quantity <= 0 {
			continue
		}

		return bid
	}

	return nil
}
