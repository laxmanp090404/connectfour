package event

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/IBM/sarama"
)

type Consumer struct {
	consumer sarama.Consumer
	topic    string
	
	// Analytics State
	totalGames    int
	totalDuration float64
	gamesByHour   map[string]int
}

func NewConsumer(brokers []string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	c, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		consumer:    c,
		topic:       "game-analytics",
		gamesByHour: make(map[string]int),
	}, nil
}

func (c *Consumer) Start() {
	partitionConsumer, err := c.consumer.ConsumePartition(c.topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Printf("ANALYTICS: Could not start consumer: %v", err)
		return
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	log.Println("ANALYTICS: Consumer started. Tracking metrics...")

	go func() {
		for {
			select {
			case msg := <-partitionConsumer.Messages():
				var event GameAnalyticsEvent
				if err := json.Unmarshal(msg.Value, &event); err == nil {
					c.processEvent(event)
				}
			case err := <-partitionConsumer.Errors():
				log.Printf("ANALYTICS ERROR: %v", err)
			case <-signals:
				return
			}
		}
	}()
}

func (c *Consumer) processEvent(e GameAnalyticsEvent) {
	// 1. Update Metrics
	c.totalGames++
	c.totalDuration += e.Duration
	
	// Track Games per Hour (Key: "2025-12-03 15:00")
	currentHour := time.Now().Format("2006-01-02 15:00")
	c.gamesByHour[currentHour]++

	// 2. Calculate Derived Stats
	avgDuration := 0.0
	if c.totalGames > 0 {
		avgDuration = c.totalDuration / float64(c.totalGames)
	}

	// 3. Log the "Report"
	log.Printf("📊 [ANALYTICS REPORT]")
	log.Printf("   Last Game:   %s (Winner: %s, Time: %.2fs)", e.GameID, e.Winner, e.Duration)
	log.Printf("   Total Games: %d", c.totalGames)
	log.Printf("   Avg Duration: %.2f seconds", avgDuration)
	log.Printf("   Games This Hour: %d", c.gamesByHour[currentHour])
	log.Printf("------------------------------------------------")
}