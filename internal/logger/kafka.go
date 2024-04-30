package logger

import (
	"github.com/segmentio/kafka-go"
)

var _ kafka.Logger = (*KafkaAdapted)(nil)

type KafkaAdapted struct{}

func (k *KafkaAdapted) Printf(_ string, _ ...any) {}

func NewKafkaAdapted() *KafkaAdapted {
	return &KafkaAdapted{}
}

func (k *KafkaAdapted) WithServiceName(_ string) *KafkaAdapted {
	return k
}

func (k *KafkaAdapted) ForErrors() *KafkaAdapted {
	return k
}
