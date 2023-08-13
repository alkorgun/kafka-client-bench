CREATE DATABASE IF NOT EXISTS faker;

CREATE TABLE IF NOT EXISTS faker.events
(
    `id` UUID,
    `status` UInt8,
    `country` Enum8('us' = 1, 'gb' = 2, 'jp' = 3),
    `dt` DateTime
)
ENGINE = GenerateRandom(1, 30);

CREATE DATABASE IF NOT EXISTS streams;

CREATE TABLE IF NOT EXISTS streams.events_producer
(
    `id` UUID,
    `status` UInt8,
    `country` String,
    `dt` DateTime,
    `_key` String
)
ENGINE = Kafka
SETTINGS  kafka_broker_list = 'kafka:29092',
    kafka_topic_list = 'events',
    kafka_group_name = 'foobar_group',
    kafka_format = 'JSONEachRow';
