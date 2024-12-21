.PHONY: all down build clean docker-build up help

# Stop and remove containers
down: ## Stop and remove containers
	docker-compose down

# Build the binary
build: ## Build the Go binary
	cd app && \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ./out/webserver .

# Clean logs
clean: ## Clean logs
	rm -rf logs/*

# Build the Docker image
docker-build: ## Build the Docker image
	docker build -t webserver .

# Start the containers
up: ## Start containers using docker-compose
	docker-compose up -d

create-kafka-topic: ## create required topic in kafka
	docker exec kafka-dev kafka-topics --create --bootstrap-server :9092 --replication-factor 1 --partitions 3 --topic events

show-events: ## get all events published to kafka
	docker exec kafka-dev kafka-console-consumer --topic events --bootstrap-server :9092 --from-beginning

# Default target
all: down build clean docker-build up create-kafka-topic ## Run all steps in sequence

help: ## Show available options
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
