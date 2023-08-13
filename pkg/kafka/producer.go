package kafka

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sirupsen/logrus"
)

const (
	CommitTimeout     = time.Second * 90
	SendRetryInterval = time.Millisecond * 200
)

type KafkaProducer struct {
	engine *kafka.Producer
	conf   KafkaProducerConfig
	logger *logrus.Logger
}

func NewKafkaProducer(logger *logrus.Logger) (*KafkaProducer, error) {
	conf := KafkaProducerConfig{}
	if err := initKafkaProducerConfig(&conf); err != nil {
		return nil, fmt.Errorf("failed to load kafka config: %s", err.Error())
	}
	engine, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers":     conf.BootstrapServers,
			"security.protocol":     conf.SecurityProtocol,
			"sasl.mechanism":        conf.SaslMechanism,
			"sasl.username":         conf.User,
			"sasl.password":         conf.Password,
			"client.id":             conf.ClientId,
			"broker.address.family": "v4",
			"message.max.bytes":     conf.MessageMaxBytes,
		},
	)
	if err == nil {
		return &KafkaProducer{
			engine: engine,
			conf:   conf,
			logger: logger,
		}, nil
	}
	return nil, fmt.Errorf("failed to create kafka producer: %s", err.Error())
}

func (k *KafkaProducer) sendSync(ch chan kafka.Event, topic string, message, key []byte) (err error) {
	err = k.engine.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: message,
		Key:   key,
	}, ch)

	if err != nil {
		if err.(kafka.Error).Code() == kafka.ErrQueueFull {
			k.logger.Error("kafka producer buffer is full")
			time.Sleep(SendRetryInterval)

			err = k.sendSync(ch, topic, message, key)
		} else {
			k.logger.Error(err)
		}
		return err
	}

	select {
	case e := <-ch:
		switch ev := e.(type) {
		case *kafka.Message:
			err = ev.TopicPartition.Error
			if err != nil {
				k.logger.Errorf("delivery failed (key=%s): %s", ev.Key, err.Error())
			}
		case kafka.Error:
			err = errors.New(ev.String())
			k.logger.Error(err)
		}
	case <-time.After(CommitTimeout):
		err = errors.New("kafka producer send timeout")
	}
	return err
}

func (k *KafkaProducer) sendAsyncGroup(
	wg *sync.WaitGroup,
	eventsCh chan kafka.Event,
	errorsCh chan error,
	topic string,
	message, key []byte,
) {
	defer wg.Done()

	if err := k.sendSync(eventsCh, topic, message, key); err != nil {
		errorsCh <- err
	}
}

func (k *KafkaProducer) SendMany(topic string, messages, keys [][]byte) (err error) {
	var wg sync.WaitGroup

	l := len(messages)
	if len(keys) != l {
		return errors.New("len(messages) != len(keys)")
	}
	eventsCh := make(chan kafka.Event, l)
	errorsCh := make(chan error, l)

	wg.Add(l)

	for i := 0; i < l; i++ {
		go k.sendAsyncGroup(&wg, eventsCh, errorsCh, topic, messages[i], keys[i])
	}

	go func() { wg.Wait(); close(errorsCh) }()

	for err := range errorsCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *KafkaProducer) Send(topic string, message, key []byte) (err error) {
	ch := make(chan kafka.Event, 3)
	return k.sendSync(ch, topic, message, key)
}

func (k *KafkaProducer) Start() {
	go func() {
		for l := range k.engine.Logs() {
			k.logger.Debugf("[%s] %s", l.Tag, l.Message)
		}
	}()
}

func (k *KafkaProducer) Stop() {
	k.engine.Close()
}
