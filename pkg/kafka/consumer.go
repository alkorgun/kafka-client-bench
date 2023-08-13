package kafka

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sirupsen/logrus"
)

const (
	StaticGroupSessionTimeoutMs = 1000 * 180
	ReadRetryInterval           = time.Millisecond * 200
)

type MsgHandler func(msg *kafka.Message) error

type BatchHandler func(batch []*kafka.Message) error

type KafkaConsumer struct {
	engine       *kafka.Consumer
	conf         KafkaConsumerConfig
	logger       *logrus.Logger
	msgHandler   MsgHandler
	batchHandler BatchHandler
	batch        []*kafka.Message
	stopped      bool
}

func NewKafkaConsumer(logger *logrus.Logger) (*KafkaConsumer, error) {
	conf := KafkaConsumerConfig{}
	if err := initKafkaConsumerConfig(&conf); err != nil {
		return nil, fmt.Errorf("failed to initialize kafka consumer config: %s", err.Error())
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("can't get hostname: %s", err.Error())
	}
	engine, err := kafka.NewConsumer(
		&kafka.ConfigMap{
			"bootstrap.servers":        conf.BootstrapServers,
			"security.protocol":        conf.SecurityProtocol,
			"sasl.mechanism":           conf.SaslMechanism,
			"sasl.username":            conf.User,
			"sasl.password":            conf.Password,
			"group.id":                 conf.GroupId,
			"group.instance.id":        hostname,
			"session.timeout.ms":       StaticGroupSessionTimeoutMs,
			"broker.address.family":    "v4",
			"message.max.bytes":        conf.MessageMaxBytes,
			"enable.auto.commit":       false,
			"enable.auto.offset.store": false,
			"auto.offset.reset":        "earliest",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %s", err.Error())
	}
	return &KafkaConsumer{
		engine:  engine,
		conf:    conf,
		logger:  logger,
		stopped: false,
	}, nil
}

func (k *KafkaConsumer) Start(topics ...string) error {
	err := k.engine.SubscribeTopics(topics, nil)
	if err != nil {
		return fmt.Errorf("failed to start subscription: %s", err.Error())
	}
	k.logger.Info("Started subscription")

	var use_batch_consumer bool
	if k.batchHandler != nil {
		k.initBatchBuffer()
		use_batch_consumer = true
	} else if k.msgHandler != nil {
		use_batch_consumer = false
	} else {
		return errors.New("neither batch handler, nor message handler is set")
	}
	go k.consume(use_batch_consumer)
	return nil
}

func (k *KafkaConsumer) initBatchBuffer() {
	k.batch = make([]*kafka.Message, 0, k.conf.BatchSize)
}

func (k *KafkaConsumer) innerBatchHandler(msg *kafka.Message) error {
	k.batch = append(k.batch, msg)
	k.engine.StoreMessage(msg)
	if len(k.batch) == k.conf.BatchSize {
		if err := k.batchHandler(k.batch); err != nil {
			return err
		}
		k.engine.Commit()
		k.initBatchBuffer()
	}
	return nil
}

func (k *KafkaConsumer) consume(use_batch_consumer bool) {
	var timeout int = int(k.conf.PollTimeout * 1000)
	for !k.stopped {
		event := k.engine.Poll(timeout)
		switch e := event.(type) {
		case *kafka.Message:
			if e.TopicPartition.Error != nil {
				k.logger.Tracef("error occured during consumer poll: %s", e.TopicPartition.Error.Error())
				time.Sleep(ReadRetryInterval)
				continue
			}
			if use_batch_consumer {
				err := k.innerBatchHandler(e)
				if err != nil {
					k.logger.Errorf("failed to process batch: %s", err.Error())
					k.Stop()
					break
				}

			} else {
				err := k.msgHandler(e)
				if err != nil {
					k.logger.Errorf("failed to process message: %s", err.Error())
					k.Stop()
					break
				}
				if _, err := k.engine.CommitMessage(e); err != nil {
					k.logger.Errorf("failed to commit offset: %s", err.Error())
				}
			}
		case kafka.Error:
			if e.Code() == kafka.ErrAllBrokersDown {
				k.Stop()

				k.logger.Fatalf("error event: %s", e.Error())
			} else {
				k.logger.Errorf("error event: %s", e.Error())
			}
		}
	}
}

func (k *KafkaConsumer) SetMsgHandler(handler MsgHandler) {
	k.msgHandler = handler
}

func (k *KafkaConsumer) SetBatchHandler(handler BatchHandler) {
	k.batchHandler = handler
}

func (k *KafkaConsumer) Stop() error {
	k.stopped = true
	return k.engine.Close()
}
