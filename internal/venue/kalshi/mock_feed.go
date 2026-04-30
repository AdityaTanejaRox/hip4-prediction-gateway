package kalshi

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
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()

	sequence := uint64(1)

	// Kalshi is cent-based. Start around 53c YES probability.
	yesBidCents := int64(53)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			receiveTs := time.Now()

			yesBidCents += int64(rand.Intn(3) - 1)

			if yesBidCents < 1 {
				yesBidCents = 1
			}
			if yesBidCents > 99 {
				yesBidCents = 99
			}

			// I am simulating a one-cent-wide book:
			//
			// YES bid = 53c
			// YES ask = 54c
			//
			// Kalshi-style NO bid that implies YES ask:
			//
			// NO bid = 100 - YES ask = 46c
			yesAskCents := yesBidCents + 1
			noBidCents := 100 - yesAskCents

			yesBidBps, err := CentsToBps(yesBidCents)
			if err != nil {
				continue
			}

			yesAskBps, err := NoBidCentsToYesAskBps(noBidCents)
			if err != nil {
				continue
			}

			bidQty := int64(300 + rand.Intn(1300))
			askQty := int64(300 + rand.Intn(1300))

			m.book.ApplySnapshot(
				[]domain.PriceLevel{
					{PriceBps: yesBidBps, Quantity: bidQty},
					{PriceBps: yesBidBps - 100, Quantity: bidQty + 250},
					{PriceBps: yesBidBps - 200, Quantity: bidQty + 500},
				},
				[]domain.PriceLevel{
					{PriceBps: yesAskBps, Quantity: askQty},
					{PriceBps: yesAskBps + 100, Quantity: askQty + 250},
					{PriceBps: yesAskBps + 200, Quantity: askQty + 500},
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
