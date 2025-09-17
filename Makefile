.PHONY: help dev dev-down dev-gateway dev-auth dev-bookings dev-logs dev-logs-gateway dev-logs-auth \
	migrate migrate-down migrate-% migrate-down-% seed test test-integration lint fmt clean build \
	docker-build k8s-up k8s-down k8s-logs db-shell redis-cli nats-cli

SHELL := /bin/sh

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_.-]+:.*?## / {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ========= Dev =========
dev: ## Start all services with docker-compose
	docker-compose up -d postgres redis nats mailpit
	@echo "Waiting for services to be ready..."
	@sleep 10
	docker-compose up --build

dev-local: ## Start services locally (not in containers)
	@echo "Starting services locally..."
	@echo "Make sure PostgreSQL, Redis, NATS are running first"
	@echo "Starting Auth Service on :8081..."
	@( cd services/auth && go run . & )
	@sleep 2
	@echo "Starting Bookings Service on :8082..."
	@( cd services/bookings && go run . & )
	@sleep 2
	@echo "Starting Gateway on :8080..."
	@( cd services/gateway && go run . )

dev-auth: ## Start only auth service locally
	@( cd services/auth && go run . )

dev-bookings: ## Start only bookings service locally
	@( cd services/bookings && go run . )

dev-gateway: ## Start only gateway service locally
	@( cd services/gateway && go run . )

dev-down: ## Stop all services
	docker-compose down

dev-logs: ## Follow logs for all services
	docker-compose logs -f

dev-logs-gateway: ## Follow gateway logs
	docker-compose logs -f gateway

dev-logs-auth: ## Follow auth service logs
	docker-compose logs -f auth-svc

# ======== DB (Goose) ========
migrate: ## Run database migrations (root + any services that exist)
	@echo "Running migrations..."
	@if [ -d "./migrations" ]; then \
		echo "Migrating root ./migrations ..."; \
		goose -dir ./migrations postgres "$(DATABASE_URL)" up; \
	else \
		echo "No root ./migrations found (skipping)"; \
	fi
	@for dir in services/*; do \
		if [ -d "$$dir/migrations" ]; then \
			echo "Migrating $$dir ..."; \
			( cd "$$dir" && goose -dir ./migrations postgres "$(DATABASE_URL)" up ); \
		fi; \
	done

migrate-down: ## Rollback database migrations (root + any services that exist)
	@echo "Rolling back migrations..."
	@if [ -d "./migrations" ]; then \
		echo "Rolling back root ./migrations ..."; \
		goose -dir ./migrations postgres "$(DATABASE_URL)" down; \
	else \
		echo "No root ./migrations found (skipping)"; \
	fi
	@for dir in services/*; do \
		if [ -d "$$dir/migrations" ]; then \
			echo "Rolling back $$dir ..."; \
			( cd "$$dir" && goose -dir ./migrations postgres "$(DATABASE_URL)" down ); \
		fi; \
	done

migrate-%: ## Run migrations for a single service, e.g. `make migrate-bookings`
	@if [ -d "services/$*/migrations" ]; then \
		echo "Migrating services/$* ..."; \
		( cd "services/$*" && goose -dir ./migrations postgres "$(DATABASE_URL)" up ); \
	else \
		echo "services/$*/migrations not found (skipping)"; \
	fi

migrate-down-%: ## Rollback migrations for a single service, e.g. `make migrate-down-bookings`
	@if [ -d "services/$*/migrations" ]; then \
		echo "Rolling back services/$* ..."; \
		( cd "services/$*" && goose -dir ./migrations postgres "$(DATABASE_URL)" down ); \
	else \
		echo "services/$*/migrations not found (skipping)"; \
	fi

seed: ## Seed database with test data
	@echo "Seeding database..."
	@go run scripts/seed/main.go

# ======== Tests / Quality ========
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# ========= Build =========
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@docker system prune -f

build: ## Build all services
	@echo "Building services..."
	@mkdir -p bin
	@for service in gateway auth bookings dispatch driver payments notify; do \
		if [ -d "services/$$service" ]; then \
			echo "Building $$service..."; \
			( cd services/$$service && go build -o ../../bin/$$service . ); \
		fi; \
	done

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@for service in gateway auth bookings dispatch driver payments notify; do \
		if [ -d "services/$$service" ]; then \
			echo "Building $$service image..."; \
			docker build -f services/$$service/Dockerfile -t luxsuv/$$service:latest .; \
		fi; \
	done

# ======== Kubernetes (kind + Helm) ========
k8s-up: ## Deploy to local Kubernetes (kind)
	@echo "Setting up local Kubernetes cluster..."
	@kind create cluster --name luxsuv || true
	@kubectl config use-context kind-luxsuv
	@echo "Installing services..."
	@for service in gateway auth bookings dispatch driver payments notify; do \
		if [ -d "deploy/helm/$$service" ]; then \
			helm upgrade --install $$service deploy/helm/$$service \
				--namespace luxsuv --create-namespace \
				--set image.tag=latest; \
		fi; \
	done

k8s-down: ## Tear down local Kubernetes cluster
	@echo "Tearing down Kubernetes cluster..."
	@kind delete cluster --name luxsuv

k8s-logs: ## Get logs from Kubernetes pods (gateway)
	@kubectl logs -f -l app=gateway -n luxsuv

# ======== Utilities ========
db-shell: ## Connect to database shell
	@docker exec -it luxsuv_postgres psql -U postgres -d luxsuv

redis-cli: ## Connect to Redis CLI
	@docker exec -it luxsuv_redis redis-cli

nats-cli: ## Connect to NATS CLI (requires nats CLI tool)
	@nats --server=$(NATS_URL) stream ls

# ======== Environment ========
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/luxsuv?sslmode=disable
REDIS_URL ?= redis://localhost:6379
NATS_URL  ?= nats://localhost:4222

export DATABASE_URL
export REDIS_URL
export NATS_URL