#!/bin/bash
# Reset Kafka topic - Truncate all messages and start fresh

echo "======================================"
echo "PME Online - Reset Kafka Topic"
echo "======================================"
echo ""

TOPIC_NAME="pme-ledger"
BOOTSTRAP_SERVER="localhost:9092"

# Auto-detect Kafka container name
KAFKA_CONTAINER=$(docker ps --format "{{.Names}}" | grep -i kafka | head -n 1)

if [ -z "$KAFKA_CONTAINER" ]; then
    echo "❌ No Kafka container is running"
    echo "   Please start it with: make docker-up"
    echo ""
    echo "Available containers:"
    docker ps --format "  - {{.Names}}"
    exit 1
fi

echo "📦 Found Kafka container: $KAFKA_CONTAINER"
echo ""

echo "⚠️  WARNING: This will delete all messages in the '$TOPIC_NAME' topic"
echo "   All order history, trades, and contracts will be lost!"
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "❌ Aborted"
    exit 0
fi

echo ""
echo "🗑️  Deleting topic '$TOPIC_NAME'..."
docker exec $KAFKA_CONTAINER kafka-topics.sh \
    --delete \
    --topic $TOPIC_NAME \
    --bootstrap-server $BOOTSTRAP_SERVER 2>/dev/null

# Wait a moment for deletion to complete
sleep 2

echo "✅ Topic deleted"
echo ""
echo "📊 Recreating topic '$TOPIC_NAME'..."
docker exec $KAFKA_CONTAINER kafka-topics.sh \
    --create \
    --topic $TOPIC_NAME \
    --bootstrap-server $BOOTSTRAP_SERVER \
    --partitions 3 \
    --replication-factor 1 \
    --if-not-exists

if [ $? -eq 0 ]; then
    echo "✅ Topic recreated successfully"
    echo ""
    echo "======================================"
    echo "Kafka topic reset complete!"
    echo "======================================"
    echo ""
    echo "Next steps:"
    echo "  1. Restart all services to reconnect"
    echo "  2. Load master data: make test-eclearapi"
    echo "  3. Start fresh with clean state"
else
    echo "❌ Failed to recreate topic"
    exit 1
fi
