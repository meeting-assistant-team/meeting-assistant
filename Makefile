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

db-sync: ## Sync production database to local (only users table)
	@echo "🔄 Syncing production database to local..."
	@echo "⚠️  This will copy users from production to local"
	@read -p "Enter production DB host: " PROD_HOST; \
	read -p "Enter production DB user: " PROD_USER; \
	read -sp "Enter production DB password: " PROD_PASS; \
	echo ""; \
	PGPASSWORD=$$PROD_PASS pg_dump -h $$PROD_HOST -U $$PROD_USER -d meeting_assistant \
		--table=users --data-only --column-inserts | \
	PGPASSWORD=postgres psql -h localhost -U postgres -d meeting_assistant
	@echo "✅ Database sync complete"

db-seed: ## Seed local database with test user
	@echo "🌱 Seeding database with test data..."
	@docker exec -i meeting-assistant-postgres psql -U postgres -d meeting_assistant << 'EOF'
	INSERT INTO users (id, email, name, oauth_provider, oauth_id, is_email_verified, is_active, created_at, updated_at)
	VALUES (
		gen_random_uuid(),
		'test@example.com',
		'Test User',
		'google',
		'test-oauth-id-123',
		true,
		true,
		NOW(),
		NOW()
	) ON CONFLICT (email) DO UPDATE SET
		name = EXCLUDED.name,
		updated_at = NOW();
	SELECT id, email, name FROM users WHERE email = 'test@example.com';
	EOF
	@echo "✅ Test user created/updated"

# Application commands
build: ## Build the application
	@echo "Building application..."
	go build -o bin/$(APP_NAME) cmd/api/main.go

run: ## Run the application
	@echo "Running application..."
	@echo "Checking for existing process on port 8080..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	air

dev-full: db-up dev ## Start database and run in development mode

killport: ## Kill process on port (usage: make killport PORT=8080)
	@echo "Killing process on port $(PORT)..."
	@lsof -ti:$(PORT) | xargs kill -9 2>/dev/null || echo "No process found on port $(PORT)"

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	swag init -g cmd/api/main.go --parseDependency --parseInternal --parseDepth 2
