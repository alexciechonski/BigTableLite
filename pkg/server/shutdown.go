package server

import (
	"context"
	"time"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)


func shutdown(grpcServer *Server) {
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	metricsServer.Shutdown(ctx)
	log.Println("Servers stopped")
}