package book

import (
	"testing"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

func TestBookTopOfBookAfterSnapshot(t *testing.T) {
	localBook := NewBook(
		domain.VenueHIP4,
		"BTC",
		"HIP4_TESTNET_BTC_OUTCOME",
		3*time.Second,
	)

	localBook.ApplySnapshot(
		[]domain.PriceLevel{
			{PriceBps: 5200, Quantity: 100},
			{PriceBps: 5300, Quantity: 200},
			{PriceBps: 5100, Quantity: 300},
		},
		[]domain.PriceLevel{
			{PriceBps: 5500, Quantity: 400},
			{PriceBps: 5400, Quantity: 500},
			{PriceBps: 5600, Quantity: 600},
		},
		42,
		time.Now(),
	)

	tob, err := localBook.TopOfBook()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tob.YesBidPriceBps != 5300 {
		t.Fatalf("best bid got %d, want 5300", tob.YesBidPriceBps)
	}

	if tob.YesBidQty != 200 {
		t.Fatalf("best bid qty got %d, want 200", tob.YesBidQty)
	}

	if tob.YesAskPriceBps != 5400 {
		t.Fatalf("best ask got %d, want 5400", tob.YesAskPriceBps)
	}

	if tob.YesAskQty != 500 {
		t.Fatalf("best ask qty got %d, want 500", tob.YesAskQty)
	}

	if tob.Sequence != 42 {
		t.Fatalf("sequence got %d, want 42", tob.Sequence)
	}
}

func TestBookApplyDeltaUpdatesLevels(t *testing.T) {
	localBook := NewBook(
		domain.VenueHIP4,
		"BTC",
		"HIP4_TESTNET_BTC_OUTCOME",
		3*time.Second,
	)

	now := time.Now()

	localBook.ApplySnapshot(
		[]domain.PriceLevel{
			{PriceBps: 5300, Quantity: 100},
		},
		[]domain.PriceLevel{
			{PriceBps: 5400, Quantity: 100},
		},
		1,
		now,
	)

	localBook.SetLevel(true, 5350, 250, 2, now)

	tob, err := localBook.TopOfBook()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tob.YesBidPriceBps != 5350 {
		t.Fatalf("best bid got %d, want 5350", tob.YesBidPriceBps)
	}

	localBook.SetLevel(true, 5350, 0, 3, now)

	tob, err = localBook.TopOfBook()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tob.YesBidPriceBps != 5300 {
		t.Fatalf("best bid got %d, want 5300 after delete", tob.YesBidPriceBps)
	}
}

func TestBookStaleWhenNoUpdates(t *testing.T) {
	localBook := NewBook(
		domain.VenueHIP4,
		"BTC",
		"HIP4_TESTNET_BTC_OUTCOME",
		1*time.Millisecond,
	)

	if !localBook.IsStale() {
		t.Fatalf("new book with no updates should be stale")
	}

	localBook.ApplySnapshot(
		[]domain.PriceLevel{{PriceBps: 5300, Quantity: 100}},
		[]domain.PriceLevel{{PriceBps: 5400, Quantity: 100}},
		1,
		time.Now(),
	)

	if localBook.IsStale() {
		t.Fatalf("book should not be stale immediately after update")
	}

	time.Sleep(5 * time.Millisecond)

	if !localBook.IsStale() {
		t.Fatalf("book should be stale after staleAfter duration")
	}
}
