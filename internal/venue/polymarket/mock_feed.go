package polymarket

import (
	"context"
	"math/rand"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

type MockFeed struct {
	book    *book.Book
	updates chan domain.TopOfBook
}

func NewMockFeed(book *book.Book) *MockFeed {
	return &MockFeed{
		book:    book,
		updates: make(chan domain.TopOfBook, 1024),
	}
}

func (m *MockFeed) Updates() <-chan domain.TopOfBook {
	return m.updates
}

func (m *MockFeed) Run(ctx context.Context) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	sequence := uint64(1)

	// Start slightly away from HIP4 mock so cross-venue edges can appear.
	mid := int64(5450)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			receiveTs := time.Now()

			mid += int64(rand.Intn(15) - 7)

			if mid < 100 {
				mid = 100
			}
			if mid > 9900 {
				mid = 9900
			}

			bid := domain.PriceBps(mid - 6)
			ask := domain.PriceBps(mid + 6)

			bidQty := int64(200 + rand.Intn(1200))
			askQty := int64(200 + rand.Intn(1200))

			m.book.ApplySnapshot(
				[]domain.PriceLevel{
					{PriceBps: bid, Quantity: bidQty},
					{PriceBps: bid - 10, Quantity: bidQty + 200},
					{PriceBps: bid - 20, Quantity: bidQty + 400},
				},
				[]domain.PriceLevel{
					{PriceBps: ask, Quantity: askQty},
					{PriceBps: ask + 10, Quantity: askQty + 200},
					{PriceBps: ask + 20, Quantity: askQty + 400},
				},
				sequence,
				receiveTs,
			)

			tob, err := m.book.TopOfBook()
			if err == nil {
				tob.ExchangeTs = receiveTs
				tob.ReceiveTs = receiveTs
				m.updates <- tob
			}

			sequence++
		}
	}
}
