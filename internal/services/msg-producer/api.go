package msgproducer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type Message struct {
	ID         types.MessageID `json:"id"`
	ChatID     types.ChatID    `json:"chatId"`
	Body       string          `json:"body"`
	FromClient bool            `json:"fromClient"`
}

func (s *Service) ProduceMessage(ctx context.Context, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	cipherText := data
	if s.cipher != nil {
		nonce, errN := s.nonceFactory(s.cipher.NonceSize())
		if errN != nil {
			return fmt.Errorf("failed to get nonce: %v", errN)
		}
		cipherText = s.cipher.Seal(nonce, nonce, data, nil)
	}

	err = s.wr.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.ChatID.String()),
		Value: cipherText,
		Time:  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %v", err)
	}

	return nil
}

func (s *Service) Close() error {
	return s.wr.Close()
}
