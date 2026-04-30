FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /out

RUN go build -o /out/hip4-node ./cmd/hip4-node
RUN go build -o /out/polymarket-node ./cmd/polymarket-node
RUN go build -o /out/kalshi-node ./cmd/kalshi-node
RUN go build -o /out/aggregator ./cmd/aggregator
RUN go build -o /out/router ./cmd/router
RUN go build -o /out/cli ./cmd/cli

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/hip4-node /usr/local/bin/hip4-node
COPY --from=builder /out/polymarket-node /usr/local/bin/polymarket-node
COPY --from=builder /out/kalshi-node /usr/local/bin/kalshi-node
COPY --from=builder /out/aggregator /usr/local/bin/aggregator
COPY --from=builder /out/router /usr/local/bin/router
COPY --from=builder /out/cli /usr/local/bin/cli

COPY configs /app/configs

CMD ["hip4-node", "--config", "/app/configs/hip4-testnet.yaml"]
