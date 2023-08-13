package service

import (
	"kafka_client/pkg/kafka"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Consumer struct {
	Topic               string `mapstructure:"INPUT_KAFKA_TOPIC"`
	BatchConsumer       bool   `mapstructure:"BATCH_CONSUMER"`
	ConsumeMessages     uint32 `mapstructure:"CONSUME_MESSAGES_TOTAL"`
	ShowEveryNthMessage uint64 `mapstructure:"SHOW_EVERY_NTH_MESSAGE"`
	startTime           time.Time
	counter             uint32
}

func (c *Consumer) increaseAndCheckCounter(cnt uint32) {
	atomic.AddUint32(&c.counter, cnt)
	if c.counter >= c.ConsumeMessages {
		logrus.Infof("%d messages consumed in %s", c.counter, time.Since(c.startTime))
		os.Exit(0)
	}
}

func (c *Consumer) handler(msg *rdkafka.Message) error {
	if uint64(msg.TopicPartition.Offset)%c.ShowEveryNthMessage == 0 {
		logrus.WithFields(logrus.Fields{
			"partition": msg.TopicPartition.Partition,
			"offset":    msg.TopicPartition.Offset,
			"key":       string(msg.Key),
			"value":     string(msg.Value),
		}).Info("got message")
	}
	c.increaseAndCheckCounter(1)
	return nil
}

func (c *Consumer) batchHandler(batch []*rdkafka.Message) error {
	for _, msg := range batch {
		if uint64(msg.TopicPartition.Offset)%c.ShowEveryNthMessage == 0 {
			logrus.WithFields(logrus.Fields{
				"partition": msg.TopicPartition.Partition,
				"offset":    msg.TopicPartition.Offset,
				"key":       string(msg.Key),
				"value":     string(msg.Value),
			}).Info("got message")
		}
	}
	c.increaseAndCheckCounter(uint32(len(batch)))
	return nil
}

func Run() {
	viper.AutomaticEnv()

	consumer := Consumer{
		startTime: time.Now(),
	}
	viper.BindEnv("INPUT_KAFKA_TOPIC")
	viper.BindEnv("BATCH_CONSUMER")
	viper.BindEnv("CONSUME_MESSAGES_TOTAL")
	viper.BindEnv("SHOW_EVERY_NTH_MESSAGE")
	err := viper.Unmarshal(&consumer)
	if err != nil {
		logrus.Fatalf("unable to get confuiguration: %s", err)
	}

	logger := logrus.StandardLogger()

	//ctx := context.Background()

	service, err := kafka.NewKafkaConsumer(logger)
	if err != nil {
		logrus.Fatal(err)
	}
	if consumer.BatchConsumer {
		service.SetBatchHandler(consumer.batchHandler)
	} else {
		service.SetMsgHandler(consumer.handler)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := service.Start(consumer.Topic); err != nil {
		logrus.Fatalf("failed to start consumer: %s", err)
	}
	defer service.Stop()

	<-sigChan

	logrus.Info("stopping the service")
}
