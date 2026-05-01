# Demo Guide

## 1. Start system

```bash
docker compose up --build
```

```text
This starts:
  hip4-node
  polymarket-node
  kalshi-node
  aggregator
  router
```

## 2. View consolidated book
```bash
go run ./cmd/cli --mode aggregator --addr localhost:50060 --market HIP4_TESTNET_BTC_OUTCOME
```

### Expected:
```text
Consolidated Book: HIP4_TESTNET_BTC_OUTCOME

YES ASKS:
  ask=0.5305 x ... venue=HIP4
  ask=0.5400 x ... venue=KALSHI
  ask=0.5460 x ... venue=POLYMARKET

YES BIDS:
  bid=0.5448 x ... venue=POLYMARKET
  bid=0.5300 x ... venue=KALSHI
  bid=0.5295 x ... venue=HIP4
```

## 3. Route BUY_YES
```bash
go run ./cmd/cli --mode route --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME --side BUY_YES --qty 100
```

### Expected:
```bash
RouteDecision: venue=HIP4 price=0.5305 qty=100 reason=lowest non-stale YES ask
```
## 4. Route SELL_YES
```bash
go run ./cmd/cli --mode route --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME --side SELL_YES --qty 100
```

### Expected:
```bash
RouteDecision: venue=POLYMARKET price=0.5448 qty=100 reason=highest non-stale YES bid
```

## 5. Stream opportunities
```bash
go run ./cmd/cli --mode opportunities --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME
```

### Expected:
```bash
Opportunity: buy=HIP4 @ 0.5305 sell=POLYMARKET @ 0.5448 gross=143 bps net=123 bps
```

## 6. Failure test
```bash
Stop one node:

docker compose stop polymarket-node

The aggregator should continue running with HIP4 and Kalshi.

Restart it:

docker compose start polymarket-node

The aggregator reconnects and Polymarket quotes reappear.
```
