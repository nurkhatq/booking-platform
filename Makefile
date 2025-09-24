.PHONY: help build start stop restart logs clean test proto migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all services
	docker-compose build

start: ## Start all services
	docker-compose up -d

stop: ## Stop all services
	docker-compose down

restart: ## Restart all services
	docker-compose restart

logs: ## View logs for all services
	docker-compose logs -f

logs-service: ## View logs for specific service (make logs-service SERVICE=api-gateway)
	docker-compose logs -f $(SERVICE)

clean: ## Clean up containers, networks, volumes, and images
	docker-compose down -v --remove-orphans
	docker system prune -f

test: ## Run tests
	go test ./...

proto: ## Generate protobuf files
	find . -name "*.proto" -exec protoc --go_out=. --go-grpc_out=. {} \;

migrate: ## Run database migrations
	docker-compose exec postgres psql -U booking_user -d booking_platform -f /docker-entrypoint-initdb.d/001_initial_schema.sql
	docker-compose exec postgres psql -U booking_user -d booking_platform -f /docker-entrypoint-initdb.d/002_admin_actions.sql

dev: ## Start development environment
	docker-compose -f docker-compose.dev.yml up -d

prod: ## Deploy to production
	chmod +x deploy.sh
	./deploy.sh

backup-db: ## Backup database
	docker-compose exec postgres pg_dump -U booking_user booking_platform > backup_$(shell date +%Y%m%d_%H%M%S).sql

restore-db: ## Restore database (BACKUP_FILE required)
	docker-compose exec -T postgres psql -U booking_user booking_platform < $(BACKUP_FILE)

health: ## Check health of all services
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health || echo "❌ API Gateway unhealthy"
	@curl -s http://localhost:8081/health || echo "❌ User Service unhealthy"
	@curl -s http://localhost:8082/health || echo "❌ Booking Service unhealthy"
	@curl -s http://localhost:8083/health || echo "❌ Notification Service unhealthy"
	@curl -s http://localhost:8084/health || echo "❌ Payment Service unhealthy"
	@curl -s http://localhost:8085/health || echo "❌ Admin Service unhealthy"

setup: ## Initial setup for development
	@echo "Setting up development environment..."
	@cp .env.example .env || echo ".env already exists"
	@echo "Please edit .env file with your configuration"
	@echo "Then run: make dev"
proto: ## Generate protobuf files
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		user-service/proto/user.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		booking-service/proto/booking.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		notification-service/proto/notification.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		payment-service/proto/payment.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		admin-service/proto/admin.proto
	@echo "✅ Proto files generated successfully!"

proto-install: ## Install protoc and Go plugins
	sudo apt update && sudo apt install -y protobuf-compiler
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Protocol Buffers tools installed!"
