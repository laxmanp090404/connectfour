package event

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/IBM/sarama"
)

type Consumer struct {
	consumer sarama.Consumer
	topic    string
}

func NewConsumer(brokers []string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	c, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: c, topic: "game-analytics"}, nil
}

func (c *Consumer) Start() {
	partitionConsumer, err := c.consumer.ConsumePartition(c.topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Printf("ANALYTICS: Could not start consumer: %v", err)
		return
	}

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	log.Println("ANALYTICS: Consumer started. Listening for game events...")

	go func() {
		for {
			select {
			case msg := <-partitionConsumer.Messages():
				var event GameAnalyticsEvent
				if err := json.Unmarshal(msg.Value, &event); err == nil {
					log.Printf("📊 ANALYTICS PROCESSED: Winner=%s GameID=%s", event.Winner, event.GameID)
				}
			case err := <-partitionConsumer.Errors():
				log.Printf("ANALYTICS ERROR: %v", err)
			case <-signals:
				return
			}
		}
	}()
}