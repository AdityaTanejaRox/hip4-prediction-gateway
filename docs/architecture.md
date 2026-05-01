# Architecture

## Core idea
```text
The system separates venue-specific integration from canonical market logic.

Each venue node owns:

  - API/WebSocket connectivity
  - venue-specific parsing
  - local book maintenance
  - venue health
  - top-of-book publication

The aggregator owns:

  - cross-venue consolidation
  - sorted YES bids
  - sorted YES asks
  - stale quote filtering metadata

The router owns:

  - route selection
  - arbitrage detection
  - execution simulation
```
---

## Service boundary

```text
Venue Node → Aggregator → Router → CLI / External API
```
---

# Architecture & Design
```text
All service-to-service communication uses **gRPC**.
```
---

## Why Not One Monolith?
```text
A distributed architecture lets each venue connector fail independently.
```

### If Polymarket Disconnects
  polymarket-node fails
  aggregator reconnects
  HIP4 and Kalshi remain alive
  router continues using healthy venues


### If HIP4 Is Stale
  HIP4 quotes are marked stale
  router avoids HIP4 for execution
  aggregator still displays it for diagnostics


---

## Canonical Book Model
```text
Every venue is normalized into:

  - YES bids
  - YES asks

This simplifies the system and enables **unified routing across all venues**.
```
---

## Venue Models

### HIP-4
```text
Native outcome market style:
  price = probability
  0.53 = 53% implied probability
```

### Polymarket
```text
Outcome-token price:
  YES token price = YES probability
  NO token price = NO probability
```

### Kalshi
```text
Kalshi-style YES/NO model:
  YES bid = internal YES bid
  NO bid = internal YES ask via (1 - NO bid)
```

---

## Router Logic

### BUY_YES
```text
choose lowest non-stale YES ask
```

### SELL_YES

```text
choose highest non-stale YES bid
```

---

## Arbitrage Logic
```text
An opportunity exists when:

  best_bid_venue != best_ask_venue
  AND best_bid_price > best_ask_price
  AND net_edge_after_costs >= threshold

```
---

## Failure Model

### Streaming Layer
```text
- Each streaming client has reconnect logic
- Aggregator automatically retries on disconnect
```

### Market Data Integrity
```text
Each book includes:

  - Stale detection
  - Timestamp tracking

Each quote carries:


  - venue
  - sequence
  - receive timestamp
  - stale flag

```

---

## Safety Guarantees
```text
This design ensures:

- Router avoids stale quotes automatically
- System continues operating during partial failures
- Aggregator provides visibility even for degraded venues
- Decisions do not rely on a single data source
```
---
