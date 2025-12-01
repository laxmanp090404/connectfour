package event

import (
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

type GameAnalyticsEvent struct {
	Event    string `json:"event"` // "GAME_OVER"
	GameID   string `json:"gameId"`
	Winner   string `json:"winner"`
	Duration int64  `json:"duration_seconds"` // Placeholder for simplicity
}

func NewProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll

	// Retry logic
	config.Producer.Retry.Max = 5

	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &Producer{producer: p, topic: "game-analytics"}, nil
}

func (p *Producer) EmitGameOver(gameID, winner string) {
	event := GameAnalyticsEvent{
		Event:  "GAME_OVER",
		GameID: gameID,
		Winner: winner,
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
		log.Printf("KAFKA: Event sent for game %s", gameID)
	}
}

func (p *Producer) Close() {
	p.producer.Close()
}