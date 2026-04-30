package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	mode := flag.String("mode", "node", "node, aggregator, route, or opportunities")
	addr := flag.String("addr", "localhost:50051", "grpc address")
	market := flag.String("market", "HIP4_TESTNET_BTC_OUTCOME", "canonical market id")
	side := flag.String("side", "BUY_YES", "BUY_YES or SELL_YES")
	qty := flag.Int64("qty", 100, "quantity")
	flag.Parse()

	switch *mode {
	case "node":
		runNodeClient(*addr, *market)
	case "aggregator":
		runAggregatorClient(*addr, *market)
	case "route":
		runRouteClient(*addr, *market, *side, *qty)
	case "opportunities":
		runOpportunityClient(*addr, *market)
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

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
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

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
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

func runRouteClient(addr string, market string, side string, qty int64) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewRouterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := client.SubmitIntent(ctx, &pb.OrderIntent{
		CanonicalMarketId: market,
		Side:              side,
		Quantity:          qty,
	})
	if err != nil {
		log.Fatalf("submit intent: %v", err)
	}

	fmt.Printf(
		"RouteDecision: venue=%s price=%0.4f qty=%d reason=%s\n",
		decision.SelectedVenue,
		float64(decision.ExpectedPriceBps)/10000.0,
		decision.ExpectedQuantity,
		decision.Reason,
	)
}

func runOpportunityClient(addr string, market string) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewRouterClient(conn)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	stream, err := client.StreamOpportunities(ctx, &pb.OpportunityRequest{
		CanonicalMarketId: market,
	})
	if err != nil {
		log.Fatalf("stream opportunities: %v", err)
	}

	fmt.Println("Waiting for opportunities...")

	for {
		opp, err := stream.Recv()
		if err != nil {
			log.Fatalf("recv opportunity: %v", err)
		}

		fmt.Printf(
			"Opportunity: buy=%s @ %0.4f sell=%s @ %0.4f gross=%d bps net=%d bps\n",
			opp.BuyVenue,
			float64(opp.BuyPriceBps)/10000.0,
			opp.SellVenue,
			float64(opp.SellPriceBps)/10000.0,
			opp.GrossEdgeBps,
			opp.NetEdgeBps,
		)
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
