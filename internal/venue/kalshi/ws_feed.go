package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type WSFeedConfig struct {
	WebSocketURL      string
	VenueMarketID     string
	CanonicalMarketID string

	APIKeyEnv    string
	APISecretEnv string

	ReconnectDelay time.Duration
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
	if w.cfg.ReconnectDelay <= 0 {
		w.cfg.ReconnectDelay = 2 * time.Second
	}

	apiKey := os.Getenv(w.cfg.APIKeyEnv)
	apiSecret := os.Getenv(w.cfg.APISecretEnv)

	if apiKey == "" || apiSecret == "" {
		return fmt.Errorf("kalshi real websocket mode requires %s and %s env vars", w.cfg.APIKeyEnv, w.cfg.APISecretEnv)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := w.runOnce(ctx, apiKey, apiSecret)
		if err != nil && ctx.Err() == nil {
			w.logger.Error().
				Err(err).
				Dur("reconnect_delay", w.cfg.ReconnectDelay).
				Msg("kalshi websocket disconnected; reconnecting")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.cfg.ReconnectDelay):
		}
	}
}

func (w *WSFeed) runOnce(ctx context.Context, apiKey string, apiSecret string) error {
	header := http.Header{}

	// Auth placeholder:
	// Kalshi authentication may require signed headers depending on the current API version.
	// Kept this adapter mock-first until exact signing is wired against the current docs.
	header.Set("KALSHI-ACCESS-KEY", apiKey)
	header.Set("KALSHI-ACCESS-SIGNATURE", apiSecret)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, w.cfg.WebSocketURL, header)
	if err != nil {
		return fmt.Errorf("dial kalshi websocket: %w", err)
	}
	defer conn.Close()

	w.logger.Info().
		Str("url", w.cfg.WebSocketURL).
		Str("market", w.cfg.VenueMarketID).
		Msg("connected to kalshi websocket")

	if err := w.subscribeOrderBook(conn); err != nil {
		return err
	}

	conn.SetReadLimit(8 << 20)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read kalshi websocket message: %w", err)
		}

		receiveTs := time.Now()

		if err := w.handleMessage(raw, receiveTs); err != nil {
			w.logger.Warn().
				Err(err).
				Str("raw", string(raw)).
				Msg("failed to handle kalshi message")
		}
	}
}

func (w *WSFeed) subscribeOrderBook(conn *websocket.Conn) error {
	request := wsSubscribeRequest{
		ID:  1,
		Cmd: "subscribe",
		Params: map[string]any{
			"channels":       []string{"orderbook_delta"},
			"market_tickers": []string{w.cfg.VenueMarketID},
		},
	}

	raw, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal kalshi subscription: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return fmt.Errorf("send kalshi orderbook subscription: %w", err)
	}

	w.logger.Info().
		Str("market", w.cfg.VenueMarketID).
		Msg("sent kalshi orderbook subscription")

	return nil
}

func (w *WSFeed) handleMessage(raw []byte, receiveTs time.Time) error {
	var envelope wsEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("unmarshal kalshi envelope: %w", err)
	}

	switch envelope.Type {
	case "subscribed":
		w.logger.Info().
			Uint64("seq", envelope.Seq).
			Msg("kalshi subscription acknowledged")
		return nil

	case "orderbook_snapshot":
		return w.handleSnapshot(envelope.Msg, envelope.Seq, receiveTs)

	case "orderbook_delta":
		return w.handleDelta(envelope.Msg, envelope.Seq, receiveTs)

	case "error":
		return fmt.Errorf("kalshi websocket error: %s", string(envelope.Msg))

	default:
		return nil
	}
}

func (w *WSFeed) handleSnapshot(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) error {
	var msg orderbookSnapshotMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("unmarshal kalshi snapshot: %w", err)
	}

	yesBids := make([]domain.PriceLevel, 0, len(msg.Yes))
	yesAsks := make([]domain.PriceLevel, 0, len(msg.No))

	for _, level := range msg.Yes {
		if len(level) < 2 {
			continue
		}

		priceBps, err := CentsToBps(level[0])
		if err != nil {
			return err
		}

		yesBids = append(yesBids, domain.PriceLevel{
			PriceBps: priceBps,
			Quantity: level[1],
		})
	}

	for _, level := range msg.No {
		if len(level) < 2 {
			continue
		}

		yesAskBps, err := NoBidCentsToYesAskBps(level[0])
		if err != nil {
			return err
		}

		yesAsks = append(yesAsks, domain.PriceLevel{
			PriceBps: yesAskBps,
			Quantity: level[1],
		})
	}

	w.book.ApplySnapshot(yesBids, yesAsks, sequence, receiveTs)
	return w.publishTopOfBook(sequence, receiveTs)
}

func (w *WSFeed) handleDelta(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) error {
	var msg orderbookDeltaMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("unmarshal kalshi delta: %w", err)
	}

	side := strings.ToLower(msg.Side)

	switch side {
	case "yes":
		priceBps, err := CentsToBps(msg.Price)
		if err != nil {
			return err
		}

		// This skeleton treats delta as absolute if positive.
		// TODO: confirm whether Kalshi delta is additive or absolute
		// and maintain level quantities accordingly.
		w.book.ApplyDelta(true, priceBps, maxInt64(0, msg.Delta), sequence, receiveTs)

	case "no":
		yesAskBps, err := NoBidCentsToYesAskBps(msg.Price)
		if err != nil {
			return err
		}

		w.book.ApplyDelta(false, yesAskBps, maxInt64(0, msg.Delta), sequence, receiveTs)

	default:
		return fmt.Errorf("unknown kalshi side: %s", msg.Side)
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
		w.logger.Warn().Msg("kalshi update channel full; dropping top-of-book update")
	}

	return nil
}

func maxInt64(a int64, b int64) int64 {
	return int64(math.Max(float64(a), float64(b)))
}
