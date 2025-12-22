package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/alexciechonski/BigTableLite/proto"
	"github.com/alexciechonski/BigTableLite/pkg/config"
	"github.com/alexciechonski/BigTableLite/pkg/sharding"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func closeConnections(conns map[int]*grpc.ClientConn) {
	for id, conn := range conns {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection for shard %d: %v", id, err)
		}
	}
}

func main() {
	operation := flag.String("op", "get", "Operation: 'set' or 'get'")
	key := flag.String("key", "test", "Key")
	value := flag.String("value", "hello", "Value (for set operation)")
	flag.Parse()

	// load config
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// create a shard map
	shardConfigPath := cfg.ShardConfigPath
	cluster, err := config.LoadCluster(shardConfigPath)

	if err != nil {
		log.Fatalf("unable to load cluster: %v", err)
	}

	shardMap, err := sharding.NewShardMap(cluster.Shards)
	if err != nil {
		log.Fatalf("unable to create a shard map: %v", err)
	}

	// create and map connections
	clients := make(map[int]proto.BigTableLiteClient)
	conns := make(map[int]*grpc.ClientConn)

	for _, sd := range cluster.Shards {
		conn, err := grpc.Dial(
			sd.Address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Fatalf("failed to connect to shard %d: %v", sd.ID, err)
		}

		clients[sd.ID] = proto.NewBigTableLiteClient(conn)
	}
	defer closeConnections(conns)

	// routing
	target := shardMap.Resolve(*key)
	client, ok := clients[target.ID]
	if !ok {
		log.Fatalf("no client found for shard %d", target.ID)
	}

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
