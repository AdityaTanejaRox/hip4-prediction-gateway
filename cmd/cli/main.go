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
	addr := flag.String("addr", "localhost:50051", "market data node address")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		CanonicalMarketIds: []string{"HIP4_TESTNET_BTC_OUTCOME"},
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
