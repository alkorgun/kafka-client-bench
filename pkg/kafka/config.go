package kafka

import (
	"github.com/spf13/viper"
)

const (
	KAFKA_MESSAGE_MAX_BYTES     = 10485760 // 10Mb
	KAFKA_PRODUCER_POLL_TIMEOUT = 0.1      // 100ms
	KAFKA_CONSUMER_POLL_TIMEOUT = 0.1
	KAFKA_CONSUMER_BATCH_SIZE   = 10_000
)

type commonConfig struct {
	BootstrapServers string `mapstructure:"KAFKA_SERVERS_STR"`
	SecurityProtocol string `mapstructure:"KAFKA_SECURITY_PROTOCOL"`
	SaslMechanism    string `mapstructure:"KAFKA_SASL_MECHANISM"`
	User             string `mapstructure:"KAFKA_USER"`
	Password         string `mapstructure:"KAFKA_PASSWORD"`
	MessageMaxBytes  int    `mapstructure:"KAFKA_MESSAGE_MAX_BYTES"`
}

type KafkaProducerConfig struct {
	ClientId    string  `mapstructure:"KAFKA_CLIENT_ID"`
	PollTimeout float32 `mapstructure:"KAFKA_PRODUCER_POLL_TIMEOUT"`
	commonConfig
}

type KafkaConsumerConfig struct {
	commonConfig
	GroupId     string  `mapstructure:"KAFKA_CONSUMER_GROUP"`
	PollTimeout float32 `mapstructure:"KAFKA_CONSUMER_POLL_TIMEOUT"`
	BatchSize   int     `mapstructure:"KAFKA_CONSUMER_BATCH_SIZE"`
}

// Bind configuration to environment variables.
func initCommonConfig(conf *commonConfig) error {
	viper.BindEnv("KAFKA_SERVERS_STR")
	viper.SetDefault("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")
	viper.SetDefault("KAFKA_SASL_MECHANISM", "")
	viper.BindEnv("KAFKA_USER")
	viper.BindEnv("KAFKA_PASSWORD")
	viper.SetDefault("KAFKA_MESSAGE_MAX_BYTES", KAFKA_MESSAGE_MAX_BYTES)
	return viper.Unmarshal(conf)
}

func initKafkaProducerConfig(conf *KafkaProducerConfig) error {
	viper.BindEnv("KAFKA_CLIENT_ID")
	viper.SetDefault("KAFKA_PRODUCER_POLL_TIMEOUT", KAFKA_PRODUCER_POLL_TIMEOUT)
	return initCommonConfig(&conf.commonConfig)
}

func initKafkaConsumerConfig(conf *KafkaConsumerConfig) error {
	if err := initCommonConfig(&conf.commonConfig); err != nil {
		return err
	}
	viper.BindEnv("KAFKA_CONSUMER_GROUP")
	viper.SetDefault("KAFKA_CONSUMER_POLL_TIMEOUT", KAFKA_CONSUMER_POLL_TIMEOUT)
	viper.SetDefault("KAFKA_CONSUMER_BATCH_SIZE", KAFKA_CONSUMER_BATCH_SIZE)
	return viper.Unmarshal(conf)
}
