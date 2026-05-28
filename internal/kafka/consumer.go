package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Consumer reads messages from Kafka
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{reader: r, topic: topic}
}

// Consume reads messages and calls handler
func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, msg map[string]interface{}) error) {
	log.Printf("kafka: consuming from %s", c.topic)

	for {
		select {
		case <-ctx.Done():
			log.Printf("kafka: stopping consumer for %s", c.topic)
			return
		default:
			m, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("kafka: read error: %v", err)
				continue
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				log.Printf("kafka: unmarshal error: %v", err)
				continue
			}

			if err := handler(ctx, msg); err != nil {
				log.Printf("kafka: handler error: %v", err)
			}
		}
	}
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
