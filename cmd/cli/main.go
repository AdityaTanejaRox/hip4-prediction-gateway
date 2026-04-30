package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	mode := flag.String("mode", "node", "node or aggregator")
	addr := flag.String("addr", "localhost:50051", "grpc address")
	market := flag.String("market", "HIP4_TESTNET_BTC_OUTCOME", "canonical market id")
	flag.Parse()

	switch *mode {
	case "node":
		runNodeClient(*addr, *market)
	case "aggregator":
		runAggregatorClient(*addr, *market)
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}

func runNodeClient(addr string, market string) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewMarketDataNodeClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	health, err := client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("health: %v", err)
	}

	fmt.Printf("Health: node=%s venue=%s alive=%v stale=%v seq=%d status=%s\n",
		health.NodeId,
		health.Venue,
		health.Alive,
		health.Stale,
		health.LastSequence,
		health.Status,
	)

	stream, err := client.StreamTopOfBook(ctx, &pb.StreamRequest{
		CanonicalMarketIds: []string{market},
	})
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			log.Fatalf("recv: %v", err)
		}

		fmt.Printf(
			"[%s] %s bid=%0.4f x %d ask=%0.4f x %d seq=%d stale=%v\n",
			update.Venue,
			update.CanonicalMarketId,
			float64(update.YesBidPriceBps)/10000.0,
			update.YesBidQty,
			float64(update.YesAskPriceBps)/10000.0,
			update.YesAskQty,
			update.Sequence,
			update.Stale,
		)
	}
}

func runAggregatorClient(addr string, market string) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewAggregatorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	health, err := client.AggregatorHealth(ctx, &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("aggregator health: %v", err)
	}

	fmt.Printf("Aggregator Health: node=%s alive=%v status=%s\n",
		health.NodeId,
		health.Alive,
		health.Status,
	)

	stream, err := client.StreamConsolidatedBook(ctx, &pb.ConsolidatedBookRequest{
		CanonicalMarketId: market,
	})
	if err != nil {
		log.Fatalf("stream consolidated book: %v", err)
	}

	for {
		book, err := stream.Recv()
		if err != nil {
			log.Fatalf("recv consolidated book: %v", err)
		}

		printConsolidatedBook(book)
	}
}

func printConsolidatedBook(book *pb.ConsolidatedBook) {
	fmt.Println()
	fmt.Printf("Consolidated Book: %s\n", book.CanonicalMarketId)

	fmt.Println("YES ASKS:")
	for _, ask := range book.YesAsks {
		fmt.Printf(
			"  ask=%0.4f x %-8d venue=%-12s stale=%v seq=%d\n",
			float64(ask.PriceBps)/10000.0,
			ask.Quantity,
			ask.Venue,
			ask.Stale,
			ask.Sequence,
		)
	}

	fmt.Println("YES BIDS:")
	for _, bid := range book.YesBids {
		fmt.Printf(
			"  bid=%0.4f x %-8d venue=%-12s stale=%v seq=%d\n",
			float64(bid.PriceBps)/10000.0,
			bid.Quantity,
			bid.Venue,
			bid.Stale,
			bid.Sequence,
		)
	}
}
