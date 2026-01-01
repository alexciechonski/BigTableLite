package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"fmt"

	"github.com/alexciechonski/BigTableLite/pkg/config"
	"github.com/alexciechonski/BigTableLite/pkg/server"
	"github.com/alexciechonski/BigTableLite/pkg/storage"
	"github.com/alexciechonski/BigTableLite/proto"
)

func CreateDataDirectory(dataBaseDir string, shardID int) (string, string, error) {
	// Create wal directories
    shardDir := fmt.Sprintf("%s/shard%d", dataBaseDir, shardID)

    // Create the shard directory (e.g., ./data/shard0)
    if err := os.MkdirAll(shardDir, 0755); err != nil {
        return "", "", err
    }

    // Define the WAL file path INSIDE that shard directory
    walFile := shardDir + "/wal.log"

	return shardDir, walFile, nil
}

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

	shardDir, walFile, err := CreateDataDirectory(cfg.DataDir, shard.ID)
	if err != nil {
		log.Fatal(err)
	}

	engine, err := storage.NewSSTableEngine(shardDir, walFile)
    if err != nil {
        log.Fatal(err)
    }

	kafkaProducer := server.NewKafkaProducer(cfg.KafkaAddress, "db-updates")

	handler := server.NewWithSSTable(engine, kafkaProducer, *shardID)

	grpcSrv := server.NewGRPCServer()
	proto.RegisterBigTableLiteServer(grpcSrv, handler)

	grpcListener, err := server.NewListener(":" + cfg.GRPCPort)
	if err != nil {
		log.Fatal(err)
	}

	metricsSrv := &http.Server{
		Addr:    ":" + cfg.MetricsPort,
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
