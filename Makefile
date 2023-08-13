.PHONY:

services := confluent-consumer confluent-batch-consumer
clickhouse-client := clickhouse-client --multiline --multiquery

start-servers:
	docker-compose up -d kafka kafka02 kafka03 clickhouse redpanda-console

events: start-servers
	-@docker-compose exec kafka kafka-topics --bootstrap-server=localhost:29092 --create --topic events --partitions=3 --replication-factor=1 > /dev/null || true
	@docker-compose exec -T clickhouse ${clickhouse-client} < sql/insert-events.sql
	docker-compose exec kafka kafka-topics --bootstrap-server=localhost:29092 --describe --topic events

connect-to-clickhouse:
	docker-compose exec clickhouse ${clickhouse-client}

open-redpanda-console:
	xdg-open http://localhost:28182/topics/events &> /dev/null &

logs:
	docker-compose logs -f --tail=300

build:
	docker-compose build --no-cache ${services}

start: start-servers
	docker-compose --compatibility up --build -d ${services}

run-confluent-consumer:
	docker-compose --compatibility up --build confluent-consumer

run-confluent-batch-consumer:
	docker-compose --compatibility up --build confluent-batch-consumer

clean:
	docker-compose down --volumes --remove-orphans
