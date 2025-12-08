package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/alexciechonski/BigTableLite/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	serverAddr := flag.String("server", "localhost:50051", "gRPC server address")
	operation := flag.String("op", "get", "Operation: 'set' or 'get'")
	key := flag.String("key", "test", "Key")
	value := flag.String("value", "hello", "Value (for set operation)")
	flag.Parse()

	// Connect to gRPC server
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewBigTableLiteClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch *operation {
	case "set":
		resp, err := client.Set(ctx, &proto.SetRequest{
			Key:   *key,
			Value: *value,
		})
		if err != nil {
			log.Fatalf("Set failed: %v", err)
		}
		fmt.Printf("Set response: Success=%v, Message=%s\n", resp.Success, resp.Message)

	case "get":
		resp, err := client.Get(ctx, &proto.GetRequest{
			Key: *key,
		})
		if err != nil {
			log.Fatalf("Get failed: %v", err)
		}
		if resp.Found {
			fmt.Printf("Get response: Found=true, Value=%s\n", resp.Value)
		} else {
			fmt.Printf("Get response: Found=false, Message=%s\n", resp.Message)
		}
	
	case "delete":
		resp, err := client.Delete(ctx, &proto.DeleteRequest{Key: *key})
		if err != nil {
			log.Fatalf("Delete failed: %v", err)
		}

		if resp.Success {
			fmt.Printf("Delete response: Success=true, Message=%s\n", resp.Message)
		} else {
			fmt.Printf("Delete response: Success=false, Message=%s\n", resp.Message)
		}

	default:
		log.Fatalf("Unknown operation: %s. Use 'set' or 'get'", *operation)
	}
}
