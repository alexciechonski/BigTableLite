package server

import (
	"context"
	"time"

	"github.com/alexciechonski/BigTableLite/pkg/storage"
	"github.com/alexciechonski/BigTableLite/proto"
	"github.com/redis/go-redis/v9"
)

type BigTableLiteServer struct {
	proto.UnimplementedBigTableLiteServer
	engine *storage.SSTableEngine
	producer *KafkaProducer
	redis  *redis.Client
	shardID int
}

func NewWithSSTable(engine *storage.SSTableEngine, producer *KafkaProducer, shardID int) *BigTableLiteServer {
	return &BigTableLiteServer{
        engine:   engine,
        producer: producer,
        shardID:  shardID,
    }
}

func NewWithRedis(redis *redis.Client) *BigTableLiteServer {
	return &BigTableLiteServer{redis: redis}
}

func (s *BigTableLiteServer) Set(ctx context.Context, req *proto.SetRequest) (*proto.SetResponse, error) {
	start := time.Now()
	defer ObserveLatency("Set", start)

	var err error
	if s.redis != nil {
		err = s.redis.Set(ctx, req.Key, req.Value, 0).Err()
	} else {
		err = s.engine.Put(req.Key, req.Value)
	}

	if s.producer != nil {
        go s.producer.PublishEvent(s.shardID, "SET", req.Key, req.Value)
    }

	if err != nil {
		IncError("Set")
		return &proto.SetResponse{Success: false, Message: err.Error()}, nil
	}

	IncSuccess("Set")
	return &proto.SetResponse{Success: true}, nil
}

func (s *BigTableLiteServer) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {
	start := time.Now()
	defer ObserveLatency("Get", start)

	if s.redis != nil {
		val, err := s.redis.Get(ctx, req.Key).Result()
		if err == redis.Nil {
			IncNotFound("Get")
			return &proto.GetResponse{Found: false}, nil
		}
		if err != nil {
			IncError("Get")
			return &proto.GetResponse{Found: false}, nil
		}
		IncSuccess("Get")
		return &proto.GetResponse{Found: true, Value: val}, nil
	}

	val, found, err := s.engine.Get(req.Key)
	if err != nil {
		IncError("Get")
		return &proto.GetResponse{Found: false}, nil
	}
	if !found {
		IncNotFound("Get")
		return &proto.GetResponse{Found: false}, nil
	}

	IncSuccess("Get")
	return &proto.GetResponse{Found: true, Value: val}, nil
}

func (s *BigTableLiteServer) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.DeleteResponse, error) {
	start := time.Now()
	defer ObserveLatency("Delete", start)

	var err error
	if s.redis != nil {
		_, err = s.redis.Del(ctx, req.Key).Result()
	} else {
		err = s.engine.Delete(req.Key)
	}

	if err != nil {
		IncError("Delete")
		return &proto.DeleteResponse{Success: false}, nil
	}

	IncSuccess("Delete")
	return &proto.DeleteResponse{Success: true}, nil
}
