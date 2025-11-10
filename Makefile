.PHONY: help build run test clean migrate-up migrate-down migrate-create docker-up docker-down db-up db-down db-reset

# Database commands
db-up: ## Start PostgreSQL and Redis with Docker Compose
	@echo "🐳 Starting PostgreSQL and Redis..."
	docker-compose up -d postgres redis
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 3
	@echo "✅ Database services are running"

db-down: ## Stop PostgreSQL and Redis
	@echo "🛑 Stopping database services..."
	docker-compose down

db-reset: ## Reset database (WARNING: This will delete all data)
	@echo "⚠️  Resetting database..."
	docker-compose down -v
	docker-compose up -d postgres redis
	@sleep 3
	@echo "✅ Database reset complete"

db-logs: ## Show database logs
	docker-compose logs -f postgres redis

# Application commands
build: ## Build the application
	@echo "Building application..."
	go build -o bin/$(APP_NAME) cmd/api/main.go

run: ## Run the application
	@echo "Running application..."
	@echo "Checking for existing process on port 8080..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	go run cmd/api/main.go

dev: ## Run in development mode (with hot reload using Air)
	@echo "Running in development mode..."
	@echo "Checking for existing process on port 8080..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	air

dev-full: db-up dev ## Start database and run in development mode

killport: ## Kill process on port (usage: make killport PORT=8080)
	@echo "Killing process on port $(PORT)..."
	@lsof -ti:$(PORT) | xargs kill -9 2>/dev/null || echo "No process found on port $(PORT)"

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	swag init -g cmd/api/main.go