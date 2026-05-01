package aggregator

import (
	"testing"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

func TestStoreMarksOldQuoteStale(t *testing.T) {
	store := NewStore(10 * time.Millisecond)

	store.Upsert(VenueTopOfBook{
		Venue:             "HIP4",
		VenueMarketID:     "BTC",
		CanonicalMarketID: "MKT",

		YesBidPriceBps: 5300,
		YesBidQty:      100,
		YesAskPriceBps: 5400,
		YesAskQty:      100,

		ReceiveTs: time.Now(),
		Sequence:  1,
		Stale:     false,
	})

	book := store.GetConsolidatedBook("MKT")

	if len(book.YesBids) != 1 {
		t.Fatalf("expected one bid")
	}

	if book.YesBids[0].Stale {
		t.Fatalf("quote should not be stale immediately after update")
	}

	time.Sleep(20 * time.Millisecond)

	book = store.GetConsolidatedBook("MKT")

	if !book.YesBids[0].Stale {
		t.Fatalf("quote should be stale after stale threshold")
	}
}

func TestStorePreservesExplicitStaleFlag(t *testing.T) {
	store := NewStore(3 * time.Second)

	store.Upsert(VenueTopOfBook{
		Venue:             "HIP4",
		VenueMarketID:     "BTC",
		CanonicalMarketID: "MKT",

		YesBidPriceBps: domain.PriceBps(5300),
		YesBidQty:      100,
		YesAskPriceBps: domain.PriceBps(5400),
		YesAskQty:      100,

		ReceiveTs: time.Now(),
		Sequence:  1,
		Stale:     true,
	})

	book := store.GetConsolidatedBook("MKT")

	if len(book.YesBids) != 1 {
		t.Fatalf("expected one bid")
	}

	if !book.YesBids[0].Stale {
		t.Fatalf("quote should remain stale when venue marks it stale")
	}
}
