#!/bin/bash

set -e

echo "Starting terraform-indexer integration test..."

# Start services
echo "Starting services with docker compose..."
docker compose up -d

# Wait for services to be ready
echo "Waiting for services to be healthy..."
sleep 30

# Check health endpoint
echo "Checking health endpoint..."
curl -f http://localhost:8080/health || { echo "Health check failed"; exit 1; }

# Check metrics endpoint  
echo "Checking metrics endpoint..."
curl -f http://localhost:8080/metrics || { echo "Metrics check failed"; exit 1; }

# Wait a bit for indexer to process some data
echo "Waiting for indexer to process data..."
sleep 60

# Check if data was inserted into database
echo "Checking database for data..."
docker compose exec -T postgres psql -U terraform_indexer -d terraform_dashboard -c "SELECT COUNT(*) FROM terraform_files;" || { echo "Database check failed"; exit 1; }
docker compose exec -T postgres psql -U terraform_indexer -d terraform_dashboard -c "SELECT COUNT(*) FROM terraform_objects;" || { echo "Database check failed"; exit 1; }

echo "Integration test passed!"

# Clean up
echo "Cleaning up..."
docker compose down -v