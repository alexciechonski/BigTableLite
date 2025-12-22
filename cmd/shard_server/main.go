package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alexciechonski/BigTableLite/pkg/config"
	"github.com/alexciechonski/BigTableLite/pkg/server"
	"github.com/alexciechonski/BigTableLite/pkg/storage"
	"github.com/alexciechonski/BigTableLite/proto"
)

func main() {
	shardID := flag.Int("shard-id", -1, "Shard ID")
	flag.Parse()

	if *shardID < 0 {
		log.Fatal("shard-id required")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	clusterCfg, err := config.LoadCluster(cfg.ShardConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	shard, err := clusterCfg.GetShardByID(*shardID)
	if err != nil {
		log.Fatal(err)
	}

	// Create wal directories
    dataDir := cfg.DataDir + "/shard" + strconv.Itoa(shard.ID)
    walDir  := cfg.WALPath // This should be the base directory, e.g., "./data/wal"

    if err := os.MkdirAll(dataDir, 0755); err != nil {
        log.Fatalf("failed to create data dir: %v", err)
    }
    if err := os.MkdirAll(walDir, 0755); err != nil {
        log.Fatalf("failed to create wal base dir: %v", err)
    }

    // Define the WAL FILE path (pointing to a file inside the walDir)
    walFile := walDir + "/shard" + strconv.Itoa(shard.ID) + ".log"

	engine, err := storage.NewSSTableEngine(dataDir, walFile)
    if err != nil {
        log.Fatal(err)
    }

	handler := server.NewWithSSTable(engine)

	grpcSrv := server.NewGRPCServer()
	proto.RegisterBigTableLiteServer(grpcSrv, handler)

	grpcListener, err := server.NewListener(shard.Address)
	if err != nil {
		log.Fatal(err)
	}

	metricsSrv := &http.Server{
		Addr:    ":" + cfg.MetricsPort + strconv.Itoa(shard.ID),
		Handler: server.MetricsHandler(),
	}

	go grpcSrv.Serve(grpcListener)
	go metricsSrv.ListenAndServe()

	log.Printf("Shard %d listening on %s", shard.ID, shard.Address)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down shard")

	grpcSrv.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	metricsSrv.Shutdown(ctx)
}
