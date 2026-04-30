package router

import (
	"fmt"
	"strings"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
)

const (
	SideBuyYes  = "BUY_YES"
	SideSellYes = "SELL_YES"
)

type Selector struct{}

func NewSelector() *Selector {
	return &Selector{}
}

func (s *Selector) SelectRoute(
	book *pb.ConsolidatedBook,
	side string,
	quantity int64,
) (*pb.RouteDecision, error) {
	if book == nil {
		return nil, fmt.Errorf("nil consolidated book")
	}

	switch strings.ToUpper(side) {
	case SideBuyYes:
		return s.selectBuyYes(book, quantity)

	case SideSellYes:
		return s.selectSellYes(book, quantity)

	default:
		return nil, fmt.Errorf("unsupported side: %s", side)
	}
}

func (s *Selector) selectBuyYes(
	book *pb.ConsolidatedBook,
	quantity int64,
) (*pb.RouteDecision, error) {
	for _, ask := range book.YesAsks {
		if ask.Stale {
			continue
		}

		if ask.Quantity <= 0 {
			continue
		}

		selectedQty := minInt64(quantity, ask.Quantity)

		return &pb.RouteDecision{
			SelectedVenue:    ask.Venue,
			ExpectedPriceBps: ask.PriceBps,
			ExpectedQuantity: selectedQty,
			Reason:           "lowest non-stale YES ask",
		}, nil
	}

	return nil, fmt.Errorf("no healthy YES ask available")
}

func (s *Selector) selectSellYes(
	book *pb.ConsolidatedBook,
	quantity int64,
) (*pb.RouteDecision, error) {
	for _, bid := range book.YesBids {
		if bid.Stale {
			continue
		}

		if bid.Quantity <= 0 {
			continue
		}

		selectedQty := minInt64(quantity, bid.Quantity)

		return &pb.RouteDecision{
			SelectedVenue:    bid.Venue,
			ExpectedPriceBps: bid.PriceBps,
			ExpectedQuantity: selectedQty,
			Reason:           "highest non-stale YES bid",
		}, nil
	}

	return nil, fmt.Errorf("no healthy YES bid available")
}

func minInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}

	return b
}
