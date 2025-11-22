package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexciechonski/BigTableLite/pkg/storage"
	"github.com/alexciechonski/BigTableLite/proto"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

var (
	// Prometheus metrics
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bigtablelite_requests_total",
			Help: "Total number of requests",
		},
		[]string{"method", "status"},
	)

	requestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bigtablelite_request_duration_seconds",
			Help:    "Request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(requestCount)
	prometheus.MustRegister(requestLatency)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// BigTableLiteServer implements the gRPC service
type BigTableLiteServer struct {
	proto.UnimplementedBigTableLiteServer
	storageEngine *storage.SSTableEngine
	redisClient   *redis.Client
	useRedis      bool
}

// NewBigTableLiteServer creates a new server instance with SSTable engine
func NewBigTableLiteServer(dataDir string) (*BigTableLiteServer, error) {
	engine, err := storage.NewSSTableEngine(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SSTable engine: %w", err)
	}

	return &BigTableLiteServer{
		storageEngine: engine,
		useRedis:      false,
	}, nil
}

// NewBigTableLiteServerWithRedis creates a new server instance with Redis backend
func NewBigTableLiteServerWithRedis(redisAddr string) (*BigTableLiteServer, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &BigTableLiteServer{
		redisClient: rdb,
		useRedis:    true,
	}, nil
}

// Set stores a key-value pair
func (s *BigTableLiteServer) Set(ctx context.Context, req *proto.SetRequest) (*proto.SetResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestLatency.WithLabelValues("Set").Observe(duration)
	}()

	var err error
	if s.useRedis {
		err = s.redisClient.Set(ctx, req.Key, req.Value, 0).Err()
	} else {
		err = s.storageEngine.Put(req.Key, req.Value)
	}

	if err != nil {
		requestCount.WithLabelValues("Set", "error").Inc()
		return &proto.SetResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to set key: %v", err),
		}, nil
	}

	requestCount.WithLabelValues("Set", "success").Inc()
	return &proto.SetResponse{
		Success: true,
		Message: "Key set successfully",
	}, nil
}

// Get retrieves a value by key
func (s *BigTableLiteServer) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestLatency.WithLabelValues("Get").Observe(duration)
	}()

	var val string
	var found bool
	var err error

	if s.useRedis {
		val, err = s.redisClient.Get(ctx, req.Key).Result()
		if err == redis.Nil {
			requestCount.WithLabelValues("Get", "not_found").Inc()
			return &proto.GetResponse{
				Found:   false,
				Value:   "",
				Message: "Key not found",
			}, nil
		} else if err != nil {
			requestCount.WithLabelValues("Get", "error").Inc()
			return &proto.GetResponse{
				Found:   false,
				Value:   "",
				Message: fmt.Sprintf("Failed to get key: %v", err),
			}, nil
		}
		found = true
	} else {
		val, found, err = s.storageEngine.Get(req.Key)
		if err != nil {
			requestCount.WithLabelValues("Get", "error").Inc()
			return &proto.GetResponse{
				Found:   false,
				Value:   "",
				Message: fmt.Sprintf("Failed to get key: %v", err),
			}, nil
		}
		if !found {
			requestCount.WithLabelValues("Get", "not_found").Inc()
			return &proto.GetResponse{
				Found:   false,
				Value:   "",
				Message: "Key not found",
			}, nil
		}
	}

	requestCount.WithLabelValues("Get", "success").Inc()
	return &proto.GetResponse{
		Found:   true,
		Value:   val,
		Message: "Key found",
	}, nil
}

func main() {
	// Support environment variables with flag defaults
	grpcPort := flag.String("grpc-port", getEnv("GRPC_PORT", "50051"), "gRPC server port")
	metricsPort := flag.String("metrics-port", getEnv("METRICS_PORT", "9090"), "Prometheus metrics port")
	dataDir := flag.String("data-dir", getEnv("DATA_DIR", "./data"), "Data directory for SSTable storage")
	useRedis := flag.Bool("use-redis", false, "Use Redis backend instead of SSTable")
	redisAddr := flag.String("redis-addr", getEnv("REDIS_ADDR", "localhost:6379"), "Redis address (only used with --use-redis)")
	flag.Parse()

	// Create server instance
	var server *BigTableLiteServer
	var err error
	if *useRedis {
		log.Println("Using Redis backend")
		server, err = NewBigTableLiteServerWithRedis(*redisAddr)
	} else {
		log.Printf("Using SSTable backend with data directory: %s", *dataDir)
		server, err = NewBigTableLiteServer(*dataDir)
	}
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%s", *grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterBigTableLiteServer(grpcServer, server)

	// Start metrics HTTP server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsListener, err := net.Listen("tcp", fmt.Sprintf(":%s", *metricsPort))
	if err != nil {
		log.Fatalf("Failed to listen on metrics port %s: %v", *metricsPort, err)
	}

	metricsServer := &http.Server{
		Handler: metricsMux,
	}

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("gRPC server listening on port %s", *grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start metrics server in a goroutine
	go func() {
		log.Printf("Metrics server listening on port %s", *metricsPort)
		if err := metricsServer.Serve(metricsListener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

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
