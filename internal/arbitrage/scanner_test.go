package arbitrage

import (
	"testing"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
)

func TestScannerDetectsCrossVenueOpportunity(t *testing.T) {
	scanner := NewScanner(ScannerConfig{
		MinNetEdgeBps:      25,
		DefaultFeeBps:      5,
		DefaultSlippageBps: 10,
	})

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesAsks: []*pb.VenueQuote{
			{
				Venue:    "HIP4",
				PriceBps: 5300,
				Quantity: 100,
				Stale:    false,
			},
		},
		YesBids: []*pb.VenueQuote{
			{
				Venue:    "POLYMARKET",
				PriceBps: 5450,
				Quantity: 100,
				Stale:    false,
			},
		},
	}

	opps := scanner.Scan(book)

	if len(opps) != 1 {
		t.Fatalf("got %d opportunities, want 1", len(opps))
	}

	opp := opps[0]

	if opp.BuyVenue != "HIP4" {
		t.Fatalf("buy venue got %s, want HIP4", opp.BuyVenue)
	}

	if opp.SellVenue != "POLYMARKET" {
		t.Fatalf("sell venue got %s, want POLYMARKET", opp.SellVenue)
	}

	if opp.GrossEdgeBps != 150 {
		t.Fatalf("gross edge got %d, want 150", opp.GrossEdgeBps)
	}

	// cost = fee*2 + slippage = 5*2 + 10 = 20
	// net = 150 - 20 = 130
	if opp.NetEdgeBps != 130 {
		t.Fatalf("net edge got %d, want 130", opp.NetEdgeBps)
	}
}

func TestScannerIgnoresSameVenueCross(t *testing.T) {
	scanner := NewScanner(ScannerConfig{
		MinNetEdgeBps:      25,
		DefaultFeeBps:      5,
		DefaultSlippageBps: 10,
	})

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesAsks: []*pb.VenueQuote{
			{Venue: "HIP4", PriceBps: 5300, Quantity: 100},
		},
		YesBids: []*pb.VenueQuote{
			{Venue: "HIP4", PriceBps: 5450, Quantity: 100},
		},
	}

	opps := scanner.Scan(book)

	if len(opps) != 0 {
		t.Fatalf("got %d opportunities, want 0 for same venue", len(opps))
	}
}

func TestScannerRequiresMinimumNetEdge(t *testing.T) {
	scanner := NewScanner(ScannerConfig{
		MinNetEdgeBps:      100,
		DefaultFeeBps:      5,
		DefaultSlippageBps: 10,
	})

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesAsks: []*pb.VenueQuote{
			{Venue: "HIP4", PriceBps: 5300, Quantity: 100},
		},
		YesBids: []*pb.VenueQuote{
			{Venue: "POLYMARKET", PriceBps: 5360, Quantity: 100},
		},
	}

	opps := scanner.Scan(book)

	if len(opps) != 0 {
		t.Fatalf("got %d opportunities, want 0 because net edge below threshold", len(opps))
	}
}

func TestScannerIgnoresStaleQuotes(t *testing.T) {
	scanner := NewScanner(ScannerConfig{
		MinNetEdgeBps:      25,
		DefaultFeeBps:      5,
		DefaultSlippageBps: 10,
	})

	book := &pb.ConsolidatedBook{
		CanonicalMarketId: "MKT",
		YesAsks: []*pb.VenueQuote{
			{Venue: "HIP4", PriceBps: 5300, Quantity: 100, Stale: true},
		},
		YesBids: []*pb.VenueQuote{
			{Venue: "POLYMARKET", PriceBps: 5450, Quantity: 100, Stale: false},
		},
	}

	opps := scanner.Scan(book)

	if len(opps) != 0 {
		t.Fatalf("got %d opportunities, want 0 because ask is stale", len(opps))
	}
}
