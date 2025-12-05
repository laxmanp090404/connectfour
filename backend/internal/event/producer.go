package event

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"os"

	"github.com/IBM/sarama"
)
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

type GameAnalyticsEvent struct {
	Event    string  `json:"event"`
	GameID   string  `json:"gameId"`
	Winner   string  `json:"winner"`
	Duration float64 `json:"duration_seconds"` // Added duration
}

func NewProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	// CHECK FOR SASL AUTH (Required for Upstash)
	if user := os.Getenv("KAFKA_USER"); user != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = user
		config.Net.SASL.Password = os.Getenv("KAFKA_PASSWORD")
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{InsecureSkipVerify: true}
	}

	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &Producer{producer: p, topic: "game-analytics"}, nil
}

func (p *Producer) EmitGameOver(gameID, winner string, duration float64) {
	event := GameAnalyticsEvent{
		Event:    "GAME_OVER",
		GameID:   gameID,
		Winner:   winner,
		Duration: duration,
	}

	val, _ := json.Marshal(event)

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(gameID),
		Value: sarama.ByteEncoder(val),
	}

	_, _, err := p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("KAFKA ERROR: Failed to send message: %v", err)
	} else {
		log.Printf("KAFKA: Event sent for game %s (%.2fs)", gameID, duration)
	}
}

func (p *Producer) Close() {
	p.producer.Close()
}