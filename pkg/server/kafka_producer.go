package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	Writer *kafka.Writer
}

type DBEvent struct {
	Timestamp int64  `json:"timestamp"`
	ShardID   int    `json:"shard_id"`
	Method    string `json:"method"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func NewKafkaProducer(brokerAddress string, topic string) *KafkaProducer {
	return &KafkaProducer{
		Writer: &kafka.Writer{
			Addr:     kafka.TCP(brokerAddress),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *KafkaProducer) PublishEvent(shardID int, method, key, value string) error {
	event := DBEvent{
		Timestamp: time.Now().Unix(),
		ShardID:   shardID,
		Method:    method,
		Key:       key,
		Value:     value,
	}

	payload, _ := json.Marshal(event)

	return p.Writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: payload,
		},
	)
}