.PHONY: help dev dev-down migrate migrate-down seed test lint fmt clean build docker-build k8s-up k8s-down

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}\' $(MAKEFILE_LIST)

# Development
dev: ## Start all services with docker-compose
	docker-compose up -d postgres redis nats mailpit
	@echo "Waiting for services to be ready..."
	@sleep 10
	docker-compose up --build

dev-down: ## Stop all services
	docker-compose down

dev-logs: ## Follow logs for all services
	docker-compose logs -f

# Database
migrate: ## Run database migrations
	@echo "Running migrations..."
	@for service in auth bookings dispatch driver payments; do \
		echo "Migrating $$service..."; \
		cd services/$$service && goose postgres "$(DATABASE_URL)\" up && cd ../..; \
	done

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	@for service in auth bookings dispatch driver payments; do \
		echo "Rolling back $$service..."; \
		cd services/$$service && goose postgres "$(DATABASE_URL)\" down && cd ../..; \
	done

seed: ## Seed database with test data
	@echo "Seeding database..."
	@go run scripts/seed/main.go

# Testing
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

# Code quality
lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Build
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@docker system prune -f

build: ## Build all services
	@echo "Building services..."
	@mkdir -p bin
	@for service in gateway auth bookings dispatch driver payments notify; do \
		echo "Building $$service..."; \
		cd services/$$service && go build -o ../../bin/$$service . && cd ../..; \
	done

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@for service in gateway auth bookings dispatch driver payments notify; do \
		echo "Building $$service image..."; \
		docker build -f services/$$service/Dockerfile -t luxsuv/$$service:latest .; \
	done

# Kubernetes
k8s-up: ## Deploy to local Kubernetes (kind)
	@echo "Setting up local Kubernetes cluster..."
	@kind create cluster --name luxsuv || true
	@kubectl config use-context kind-luxsuv
	@echo "Installing services..."
	@for service in gateway auth bookings dispatch driver payments notify; do \
		helm upgrade --install $$service deploy/helm/$$service \
			--namespace luxsuv --create-namespace \
			--set image.tag=latest; \
	done

k8s-down: ## Tear down local Kubernetes cluster
	@echo "Tearing down Kubernetes cluster..."
	@kind delete cluster --name luxsuv

k8s-logs: ## Get logs from Kubernetes pods
	@kubectl logs -f -l app=gateway -n luxsuv

# Utilities
db-shell: ## Connect to database shell
	@docker exec -it luxsuv_postgres psql -U postgres -d luxsuv

redis-cli: ## Connect to Redis CLI
	@docker exec -it luxsuv_redis redis-cli

nats-cli: ## Connect to NATS CLI (requires nats CLI tool)
	@nats --server=nats://localhost:4222 stream ls

# Environment variables
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/luxsuv?sslmode=disable
REDIS_URL ?= redis://localhost:6379
NATS_URL ?= nats://localhost:4222

# Export environment variables
export DATABASE_URL
export REDIS_URL
export NATS_URL