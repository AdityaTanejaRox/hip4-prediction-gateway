# HIP4 Prediction Gateway

A Go-based distributed systems project for HIP-4 / Hyperliquid-style prediction market integration.

The system models a multi-venue prediction-market gateway with:

- HIP-4 / Hyperliquid-first architecture
- gRPC-based distributed venue nodes
- local order book maintenance
- normalized YES/NO pricing
- Kalshi-style YES/NO conversion
- Polymarket-style outcome price normalization
- consolidated cross-venue book
- arbitrage scanner
- smart routing simulator
- node liveness and reconnect loops
- Docker Compose demo

---

## Why this exists

Prediction markets are fragmented across venues such as:

- Hyperliquid HIP-4 / outcome markets
- Polymarket
- Kalshi

Each venue has different APIs, market identifiers, price formats, order book semantics, and settlement constraints.

This project builds the infrastructure layer needed to normalize those venues into one internal model.

---

## Architecture

```text
                     ┌──────────────┐
                     │  hip4-node   │
                     │ Hyperliquid  │
                     └──────┬───────┘
                            │ gRPC
                            ▼
┌────────────────┐    ┌──────────────┐   ┌─────────────┐
│polymarket-node │──▶│  aggregator  │◀──│ kalshi-node │
└────────────────┘    └──────┬───────┘   └─────────────┘
                            │ gRPC
                            ▼
                     ┌──────────────┐
                     │    router    │
                     └──────┬───────┘
                            │ gRPC
                            ▼
                     ┌──────────────┐
                     │     cli      │
                     └──────────────┘
```

## Services
## hip4-node

The HIP-4-first venue node.

**Responsibilities:**
- Connect to Hyperliquid testnet or mock feed
- Subscribe to CLOB-style market data
- Normalize prices into internal probability bps
- Maintain local YES order book
- Expose top-of-book over gRPC

---

## polymarket-node

Polymarket-style venue node.

**Responsibilities:**
- Emit Polymarket-style 0–1 outcome prices
- Maintain local YES order book
- Expose top-of-book over gRPC

---

## kalshi-node

Kalshi-style venue node.

**Responsibilities:**
- Model Kalshi YES/NO market structure
- Convert NO bids into YES asks
- Maintain normalized YES book
- Expose top-of-book over gRPC

---

## aggregator

Distributed book aggregator.

**Responsibilities:**
- Connect to venue nodes over gRPC
- Consume top-of-book streams
- Maintain latest quote per venue
- Build consolidated YES bid/ask book

---

## router

Routing and arbitrage service.

**Responsibilities:**
- Subscribe to consolidated book
- Route BUY_YES to lowest healthy ask
- Route SELL_YES to highest healthy bid
- Detect cross-venue probability dislocations

---

# Internal Price Model

All venues normalize into **basis points (bps) of probability**:

  0.0000 -> 0
  0.5300 -> 5300
  1.0000 -> 10000


### Examples

```bash
HIP-4 price: 0.5321 -> 5321 bps
Polymarket price: 0.54 -> 5400 bps
Kalshi cents: 53c -> 5300 bps
```

### Kalshi NO bid conversion

```bash
NO bid = 47c
YES ask = 100c - 47c = 53c
YES ask = 5300 bps
```

---

# Run Locally

### Terminal 1 — HIP4 Node

```bash
go run ./cmd/hip4-node --config configs/hip4-testnet.yaml
```

### Terminal 2 — Polymarket Node
```bash
go run ./cmd/polymarket-node --config configs/polymarket.yaml
```

### Terminal 3 — Kalshi Node
```bash
go run ./cmd/kalshi-node --config configs/kalshi.yaml
```

### Terminal 4 — Aggregator
```bash
go run ./cmd/aggregator --config configs/aggregator.yaml
```

### Terminal 5 — Router
```bash
go run ./cmd/router --config configs/router.yaml
```

### CLI Usage
### View Consolidated Book
```bash
go run ./cmd/cli --mode aggregator --addr localhost:50060 --market HIP4_TESTNET_BTC_OUTCOME
```

### Route BUY_YES
```bash
go run ./cmd/cli --mode route --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME --side BUY_YES --qty 100
```

### Route SELL_YES
```bash
go run ./cmd/cli --mode route --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME --side SELL_YES --qty 100
```

### Stream Opportunities
```bash
go run ./cmd/cli --mode opportunities --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME
```

### Run with Docker Compose
```bash
docker compose up --build
```

### Then:
```bash
go run ./cmd/cli --mode aggregator --addr localhost:50060 --market HIP4_TESTNET_BTC_OUTCOME
```
---

### Tests
```bash
go test ./...
```

### Coverage Includes
  Kalshi YES/NO normalization
  Local order book top-of-book
  Stale book detection
  Router venue selection
  Arbitrage scanner correctness
  Example Output
  Consolidated Book
  Consolidated Book: HIP4_TESTNET_BTC_OUTCOME

```bash
YES ASKS:
  ask=0.5305 x 700      venue=HIP4         stale=false seq=44
  ask=0.5400 x 900      venue=KALSHI       stale=false seq=31
  ask=0.5460 x 1200     venue=POLYMARKET   stale=false seq=39

YES BIDS:
  bid=0.5448 x 1100     venue=POLYMARKET   stale=false seq=39
  bid=0.5300 x 800      venue=KALSHI       stale=false seq=31
  bid=0.5295 x 600      venue=HIP4         stale=false seq=44
Route Decision
RouteDecision: venue=HIP4 price=0.5305 qty=100 reason=lowest non-stale YES ask
Arbitrage Opportunity
Opportunity: buy=HIP4 @ 0.5305 sell=POLYMARKET @ 0.5448 gross=143 bps net=123 bps
```

### What Is Implemented
  Distributed Go services
  gRPC streaming
  HIP-4-first market node
  Hyperliquid testnet config support
  Mockable venue feeds
  Local order books
  Consolidated book
  Router simulator
  Cross-venue opportunity scanner
  Docker Compose deployment
  Correctness tests

### Future Work
  Full real HIP-4 outcome market integration
  Real Polymarket WebSocket adapter
  Real Kalshi authenticated WebSocket adapter
  Persistent event log
  Order execution adapters
  Risk checks
  Prometheus metrics
  Web dashboard
