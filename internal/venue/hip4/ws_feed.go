package hip4

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/book"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type WSFeedConfig struct {
	WebSocketURL 				string
	Asset 							string
	VenueMarketID  			string
	CanonicalMarketID   string
	ReconnectDelay 			time.Duration
}

type WSFeed struct {
	cfg 				 WSFeedConfig
	book 				 *book.Book
	logger 			 zerolog.Logger
	updates chan domain.TopOfBook
}

func NewWSFeed(
	cfg WSFeedConfig,
	book *book.Book,
	logger zerolog.Logger,
) *WSFeed {
	return &WSFeed{
		cfg: 				cfg,
		book: 			book,
		logger: 		logger,
		updates:    make(chan domain.TopOfBook, 4096),
	}
}

func (w *WSFeed) Updates() <-chan domain.TopOfBook {
	return w.updates
}

func (w *WSFeed) Run(ctx context.Context) error {
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
				Msg("hip4 websocket disconnected; reconnectiing")
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

	header := http.Header{}

	conn, _, err := dialer.DialContext(ctx, w.cfg.WebSocketURL, header)
	if err != nil {
		return fmt.Errorf("dial hyperliquid websocket: %w", err)
	}
	defer conn.Close()

	w.logger.Info().
		Str("url", w.cfg.WebSocketURL).
		Str("asset", w.cfg.Asset).
		Msg("Connected to hyperliquid websocket")

	if err := w.subscribeL2Book(conn); err != nil {
		return err
	}

	conn.SetReadLimit(8 << 20)

	pongWait := 60 * time.Second
	pingPeriod := 20 * time.Second

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(appData string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = conn.WriteControl(
						websocket.PingMessage,
						[]byte("ping"),
						time.Now().Add(5*time.Second),
				)
			}
		}
	}()

	sequence := uint64(1)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read websocket message: %w", err)
		}

		receiveTs := time.Now()

		handled, err := w.handleMessage(raw, sequence, receiveTs)
		if err != nil {
			w.logger.Warn().
				Err(err).
				Str("raw", string(raw)).
				Msg("failed to handle hyperliquid message")
			continue
		}

		if handled {
			sequence++
		}
	}
}

func (w *WSFeed) subscribeL2Book(conn *websocket.Conn) error {
	request := subscriptionRequest{
		Method: "subscribe",
		Subscription: subscription{
			Type: "l2Book",
			Coin: w.cfg.Asset,
		},
	}

	raw, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal subscription: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return fmt.Errorf("send l2Book subscription: %w", err)
	}

	w.logger.Info().
		Str("asset", w.cfg.Asset).
		Msg("sent l2Book subscription")

	return nil
}

func (w *WSFeed) handleMessage(
	raw []byte,
	sequence uint64,
	receiveTs time.Time,
) (bool, error) {
	var generic struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(raw, &generic); err != nil {
		return false, fmt.Errorf("unmarshal generic envelope: %w", err)
	}

	switch generic.Channel {
	case "subscriptionResponse":
		w.logger.Info().
			Str("data", string(generic.Data)).
			Msg("subscription acknowledged")
		return false, nil

	case "l2Book":
		return true, w.handleL2Book(generic.Data, sequence, receiveTs)

	default:
		return false, nil
	}
}

func (w *WSFeed) handleL2Book(
	raw json.RawMessage,
	sequence uint64,
	receiveTs time.Time,
) error {
	var data hyperliquidData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("unmarshal l2Book data: %w", err)
	}

	if data.Coin != "" && data.Coin != w.cfg.Asset {
		return nil
	}

	if len(data.Levels) < 2 {
		return errors.New("l2Book data missing bid/ask levels")
	}

	bids, err := convertHyperliquidLevelsToOutcomeBps(data.Levels[0])
	if err != nil {
		return fmt.Errorf("convert bids: %w", err)
	}

	asks, err := convertHyperliquidLevelsToOutcomeBps(data.Levels[1])
	if err != nil {
		return fmt.Errorf("convert asks: %w", err)
	}

	w.book.ApplySnapshot(bids, asks, sequence, receiveTs)

	tob, err := w.book.TopOfBook()
	if err != nil {
		return err
	}

	tob.ExchangeTs = hyperliquidMillisToTime(data.Time)
	tob.ReceiveTs = receiveTs

	select {
	case w.updates <- tob:
	default:
		w.logger.Warn().Msg("hip4 update channel full; dropping top-of-book update")
	}

	return nil
}

func convertHyperliquidLevelsToOutcomeBps(
	levels []l2BookLevel,
) ([]domain.PriceLevel, error) {
	out := make([]domain.PriceLevel, 0, len(levels))

	for _, level := range levels {
		price, err := strconv.ParseFloat(level.Px, 64)
		if err != nil {
			return nil, fmt.Errorf("parse price %q: %w", level.Px, err)
		}

		size, err := strconv.ParseFloat(level.Sz, 64)
		if err != nil {
			return nil, fmt.Errorf("parse size %q: %w", level.Sz, err)
		}

		priceBps, err := normalizeHyperliquidPriceToOutcomeBps(price)
		if err != nil {
			return nil, err
		}

		quantity := int64(math.Round(size * 10000.0))
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

func normalizeHyperliquidPriceToOutcomeBps(price float64) (domain.PriceBps, error) {
	// HIP-4 outcome contracts should naturally trade in [0, 1].
	// However, while testing against regular Hyperliquid assets like BTC,
	// the raw price is not a probability. To keep the pipeline runnable,
	// we map the observed market price into a bounded synthetic probability.
	//
	// When a real HIP-4 outcome asset is available, replace this with:
	//     return normalize.ProbabilityFloatToBps(price)
	if price >= 0.0 && price <= 1.0 {
		return domain.PriceBps(math.Round(price * 10000.0)), nil
	}

	// Temporary synthetic conversion for non-HIP4 testnet assets:
	// compress large prices into a stable probability-like range.
	normalized := 0.5 + 0.4*math.Tanh((price-50000.0)/50000.0)

	if normalized < 0.0 {
		normalized = 0.0
	}
	if normalized > 1.0 {
		normalized = 1.0
	}

	return domain.PriceBps(math.Round(normalized * 10000.0)), nil
}

func hyperliquidMillisToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}

	return time.Unix(0, ms*int64(time.Millisecond))
}
