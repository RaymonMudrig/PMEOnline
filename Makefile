.PHONY: help docker-up docker-down kafka-topic run-eclearapi test-eclearapi clean build

help:
	@echo "PME Online - Makefile Commands"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make docker-up       - Start Kafka and PostgreSQL containers"
	@echo "  make docker-down     - Stop all containers"
	@echo "  make kafka-topic     - Create Kafka topic (run after docker-up)"
	@echo ""
	@echo "Services:"
	@echo "  make run-eclearapi   - Run eClear API service"
	@echo "  make run-pmeoms      - Run OMS service"
	@echo "  make run-pmeapi      - Run APME API service"
	@echo "  make run-dbexporter  - Run Database Exporter service"
	@echo ""
	@echo "Testing:"
	@echo "  make test-eclearapi  - Test eClear API with sample data"
	@echo "  make test-pmeapi     - Test APME API with sample requests"
	@echo "  make test            - Run all tests"
	@echo ""
	@echo "Reset & Cleanup:"
	@echo "  make reset-kafka     - Truncate Kafka topic (delete all messages)"
	@echo "  make reset-db        - Reset PostgreSQL database"
	@echo "  make reset-all       - Reset both Kafka and database"
	@echo ""
	@echo "Build:"
	@echo "  make build           - Build all services"
	@echo "  make clean           - Clean build artifacts"

# Infrastructure
docker-up:
	@echo "🚀 Starting Docker containers..."
	docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ Docker containers started"

docker-down:
	@echo "🛑 Stopping Docker containers..."
	docker-compose down
	@echo "✅ Docker containers stopped"

kafka-topic:
	@echo "📊 Creating Kafka topic..."
	@KAFKA_CONTAINER=$$(docker ps --format "{{.Names}}" | grep -i kafka | head -n 1); \
	if [ -z "$$KAFKA_CONTAINER" ]; then \
		echo "❌ No Kafka container is running"; \
		exit 1; \
	fi; \
	docker exec $$KAFKA_CONTAINER kafka-topics.sh \
		--create \
		--topic pme-ledger \
		--bootstrap-server localhost:9092 \
		--partitions 3 \
		--replication-factor 1 \
		--if-not-exists
	@echo "✅ Kafka topic created"

# Services
run-eclearapi:
	@echo "🌐 Starting eClear API Service..."
	cd cmd/eclearapi && go run main.go

run-pmeoms:
	@echo "⚙️  Starting OMS Service..."
	cd cmd/pmeoms && go run main.go

run-pmeapi:
	@echo "🌐 Starting APME API Service..."
	cd cmd/pmeapi && go run main.go

run-dbexporter:
	@echo "💾 Starting Database Exporter Service..."
	cd cmd/dbexporter && go run main.go

# Testing
test-eclearapi:
	@echo "🧪 Testing eClear API Service..."
	@cd tests/eclearapi && ./test.sh

test-pmeapi:
	@echo "🧪 Testing APME API Service..."
	@echo "📝 Make sure APME API service is running (make run-pmeapi)"
	@sleep 2
	@cd cmd/pmeapi && ./test_pmeapi.sh

test:
	@echo "🧪 Running all tests..."
	go test -v ./...

# Build
build:
	@echo "Building all services..."
	@mkdir -p bin
	@echo "  Building eclearapi..."
	@go build -o bin/eclearapi cmd/eclearapi/main.go
	@echo "  Building pmeoms..."
	@go build -o bin/pmeoms cmd/pmeoms/main.go
	@echo "  Building pmeapi..."
	@go build -o bin/pmeapi cmd/pmeapi/main.go
	@echo "  Building dbexporter..."
	@go build -o bin/dbexporter cmd/dbexporter/main.go
	@echo "Build complete"

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✅ Clean complete"

# Reset and cleanup
reset-kafka:
	@echo "🔄 Resetting Kafka topic..."
	@./scripts/reset-kafka.sh

reset-db:
	@echo "🔄 Resetting PostgreSQL database..."
	@POSTGRES_CONTAINER=$$(docker ps --format "{{.Names}}" | grep -i postgres | head -n 1); \
	if [ -z "$$POSTGRES_CONTAINER" ]; then \
		echo "❌ No PostgreSQL container is running"; \
		exit 1; \
	fi; \
	echo "📦 Found PostgreSQL container: $$POSTGRES_CONTAINER"; \
	docker exec $$POSTGRES_CONTAINER psql -U pmeuser -d pmedb -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "✅ Database reset complete"

reset-all: reset-kafka reset-db
	@echo "✅ Full system reset complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Restart all services"
	@echo "  2. Load master data: make test-eclearapi"

# Development workflow
setup: docker-up kafka-topic
	@echo "✅ Development environment ready!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make run-eclearapi' in one terminal"
	@echo "  2. Run 'make test-eclearapi' in another terminal to populate master data"

# Quick start for eClear API testing
quickstart: setup
	@echo "⏳ Waiting for services to stabilize..."
	@sleep 5
	@echo "🚀 Starting eClear API Service in background..."
	@cd cmd/eclearapi && go run main.go > /tmp/eclearapi.log 2>&1 &
	@echo "⏳ Waiting for API to start..."
	@sleep 3
	@echo "🧪 Running tests..."
	@make test-eclearapi
