package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/normalize"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type WSFeedConfig struct {
	WebSocketURL      string
	AssetIDs          []string
	VenueMarketID     string
	CanonicalMarketID string
	ReconnectDelay    time.Duration
}

type WSFeed struct {
	cfg     WSFeedConfig
	book    *book.Book
	logger  zerolog.Logger
	updates chan domain.TopOfBook
}

func NewWSFeed(
	cfg WSFeedConfig,
	book *book.Book,
	logger zerolog.Logger,
) *WSFeed {
	return &WSFeed{
		cfg:     cfg,
		book:    book,
		logger:  logger,
		updates: make(chan domain.TopOfBook, 4096),
	}
}

func (w *WSFeed) Updates() <-chan domain.TopOfBook {
	return w.updates
}

func (w *WSFeed) Run(ctx context.Context) error {
	if len(w.cfg.AssetIDs) == 0 {
		return fmt.Errorf("polymarket real websocket mode requires at least one asset_id")
	}

	if w.cfg.ReconnectDelay <= 0 {
		w.cfg.ReconnectDelay = 2 * time.Second
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := w.runOnce(ctx)
		if err != nil && ctx.Err() == nil {
			w.logger.Error().
				Err(err).
				Dur("reconnect_delay", w.cfg.ReconnectDelay).
				Msg("polymarket websocket disconnected; reconnecting")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.cfg.ReconnectDelay):
		}
	}
}

func (w *WSFeed) runOnce(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, w.cfg.WebSocketURL, http.Header{})
	if err != nil {
		return fmt.Errorf("dial polymarket websocket: %w", err)
	}
	defer conn.Close()

	w.logger.Info().
		Str("url", w.cfg.WebSocketURL).
		Strs("asset_ids", w.cfg.AssetIDs).
		Msg("connected to polymarket websocket")

	if err := w.subscribe(conn); err != nil {
		return err
	}

	conn.SetReadLimit(8 << 20)

	sequence := uint64(1)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read polymarket websocket message: %w", err)
		}

		receiveTs := time.Now()

		handled, err := w.handleMessage(raw, sequence, receiveTs)
		if err != nil {
			w.logger.Warn().
				Err(err).
				Str("raw", string(raw)).
				Msg("failed to handle polymarket message")
			continue
		}

		if handled {
			sequence++
		}
	}
}

func (w *WSFeed) subscribe(conn *websocket.Conn) error {
	request := wsSubscriptionRequest{
		AssetsIDs: w.cfg.AssetIDs,
		Type:      "market",
	}

	raw, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal polymarket subscription: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return fmt.Errorf("send polymarket subscription: %w", err)
	}

	w.logger.Info().Msg("sent polymarket market subscription")
	return nil
}

func (w *WSFeed) handleMessage(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) (bool, error) {
	var arrayPayload []json.RawMessage
	if err := json.Unmarshal(raw, &arrayPayload); err == nil {
		handledAny := false

		for _, item := range arrayPayload {
			handled, err := w.handleSingleMessage(item, sequence, receiveTs)
			if err != nil {
				return handledAny, err
			}
			if handled {
				handledAny = true
			}
		}

		return handledAny, nil
	}

	return w.handleSingleMessage(raw, sequence, receiveTs)
}

func (w *WSFeed) handleSingleMessage(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) (bool, error) {
	var envelope wsEventEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false, fmt.Errorf("unmarshal polymarket envelope: %w", err)
	}

	switch envelope.EventType {
	case "book":
		return true, w.handleBook(raw, sequence, receiveTs)

	case "price_change":
		return true, w.handlePriceChange(raw, sequence, receiveTs)

	default:
		return false, nil
	}
}

func (w *WSFeed) handleBook(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) error {
	var event bookEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return fmt.Errorf("unmarshal polymarket book event: %w", err)
	}

	bids, err := convertPolymarketLevels(event.Bids)
	if err != nil {
		return fmt.Errorf("convert polymarket bids: %w", err)
	}

	asks, err := convertPolymarketLevels(event.Asks)
	if err != nil {
		return fmt.Errorf("convert polymarket asks: %w", err)
	}

	w.book.ApplySnapshot(bids, asks, sequence, receiveTs)

	return w.publishTopOfBook(sequence, receiveTs)
}

func (w *WSFeed) handlePriceChange(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) error {
	var event priceChangeEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return fmt.Errorf("unmarshal polymarket price change event: %w", err)
	}

	for _, change := range event.Changes {
		priceBps, err := normalize.DecimalStringToBps(change.Price)
		if err != nil {
			return err
		}

		sizeFloat, err := strconv.ParseFloat(change.Size, 64)
		if err != nil {
			return fmt.Errorf("parse polymarket size %q: %w", change.Size, err)
		}

		quantity := int64(math.Round(sizeFloat * 10000.0))

		isBid := strings.EqualFold(change.Side, "BUY") || strings.EqualFold(change.Side, "BID")
		w.book.SetLevel(isBid, priceBps, quantity, sequence, receiveTs)
	}

	return w.publishTopOfBook(sequence, receiveTs)
}

func (w *WSFeed) publishTopOfBook(sequence uint64, receiveTs time.Time) error {
	tob, err := w.book.TopOfBook()
	if err != nil {
		return err
	}

	tob.ExchangeTs = receiveTs
	tob.ReceiveTs = receiveTs
	tob.Sequence = sequence

	select {
	case w.updates <- tob:
	default:
		w.logger.Warn().Msg("polymarket update channel full; dropping top-of-book update")
	}

	return nil
}

func convertPolymarketLevels(levels []priceLevel) ([]domain.PriceLevel, error) {
	out := make([]domain.PriceLevel, 0, len(levels))

	for _, level := range levels {
		priceBps, err := normalize.DecimalStringToBps(level.Price)
		if err != nil {
			return nil, err
		}

		sizeFloat, err := strconv.ParseFloat(level.Size, 64)
		if err != nil {
			return nil, fmt.Errorf("parse polymarket size %q: %w", level.Size, err)
		}

		quantity := int64(math.Round(sizeFloat * 10000.0))
		if quantity <= 0 {
			continue
		}

		out = append(out, domain.PriceLevel{
			PriceBps: priceBps,
			Quantity: quantity,
		})
	}

	return out, nil
}
