package router

import (
	"testing"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
)

func TestSelectBuyYesChoosesLowestHealthyAsk(t *testing.T) {
	selector := NewSelector()

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesAsks: []*pb.VenueQuote{
			{
				Venue:    "HIP4",
				PriceBps: 5300,
				Quantity: 100,
				Stale:    true,
			},
			{
				Venue:    "KALSHI",
				PriceBps: 5350,
				Quantity: 200,
				Stale:    false,
			},
			{
				Venue:    "POLYMARKET",
				PriceBps: 5400,
				Quantity: 300,
				Stale:    false,
			},
		},
	}

	decision, err := selector.SelectRoute(book, SideBuyYes, 150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.SelectedVenue != "KALSHI" {
		t.Fatalf("selected venue got %s, want KALSHI", decision.SelectedVenue)
	}

	if decision.ExpectedPriceBps != 5350 {
		t.Fatalf("price got %d, want 5350", decision.ExpectedPriceBps)
	}

	if decision.ExpectedQuantity != 150 {
		t.Fatalf("quantity got %d, want 150", decision.ExpectedQuantity)
	}
}

func TestSelectSellYesChoosesHighestHealthyBid(t *testing.T) {
	selector := NewSelector()

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesBids: []*pb.VenueQuote{
			{
				Venue:    "HIP4",
				PriceBps: 5500,
				Quantity: 100,
				Stale:    true,
			},
			{
				Venue:    "KALSHI",
				PriceBps: 5450,
				Quantity: 80,
				Stale:    false,
			},
			{
				Venue:    "POLYMARKET",
				PriceBps: 5400,
				Quantity: 300,
				Stale:    false,
			},
		},
	}

	decision, err := selector.SelectRoute(book, SideSellYes, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.SelectedVenue != "KALSHI" {
		t.Fatalf("selected venue got %s, want KALSHI", decision.SelectedVenue)
	}

	if decision.ExpectedPriceBps != 5450 {
		t.Fatalf("price got %d, want 5450", decision.ExpectedPriceBps)
	}

	if decision.ExpectedQuantity != 80 {
		t.Fatalf("quantity got %d, want 80 because venue only has 80", decision.ExpectedQuantity)
	}
}

func TestSelectRouteReturnsErrorWhenNoHealthyVenue(t *testing.T) {
	selector := NewSelector()

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesAsks: []*pb.VenueQuote{
			{
				Venue:    "HIP4",
				PriceBps: 5300,
				Quantity: 100,
				Stale:    true,
			},
		},
	}

	_, err := selector.SelectRoute(book, SideBuyYes, 100)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
