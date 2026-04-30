package hip4

import (
	"context"
	"math/rand"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

type MockFeed struct {
	book     *book.Book
	updates chan domain.TopOfBook
}

func NewMockFeed(book *book.Book) *MockFeed {
	return &MockFeed{
		book:     book,
		updates: make(chan domain.TopOfBook, 1024),
	}
}

func (m *MockFeed) Updates() <-chan domain.TopOfBook {
	return m.updates
}

func (m *MockFeed) Run(ctx context.Context) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	sequence := uint64(1)
	mid := int64(5300)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			receiveTs := time.Now()

			mid += int64(rand.Intn(11) - 5)

			if mid < 100 {
				mid = 100
			}
			if mid > 9900 {
				mid = 9900
			}

			bid := domain.PriceBps(mid - 5)
			ask := domain.PriceBps(mid + 5)

			bidQty := int64(100 + rand.Intn(900))
			askQty := int64(100 + rand.Intn(900))

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
				m.updates <- tob
			}

			sequence++
		}
	}
}
