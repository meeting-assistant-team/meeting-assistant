.PHONY: help build run test clean migrate-up migrate-down migrate-create docker-up docker-down docker-build docker-rebuild db-up db-down db-reset

APP_NAME=meeting-assistant

docker-logs-app: ## Show app logs only
	docker-compose logs -f app

# Database commands
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
build: ## Build Go binary (local build)
	@echo "📦 Building Go binary..."
	go build -o bin/$(APP_NAME) cmd/api/main.go
	@echo "✅ Build complete: ./bin/$(APP_NAME)"

build-docker: docker-build docker-up ## Build Docker image and start containers

build-docker-rebuild: docker-rebuild docker-up ## Rebuild Docker image (no cache) and start

build-docker-clean: docker-prune build-docker-rebuild ## Clean Docker resources then rebuild

run: ## Run the application
	@echo "Running application locally..."
	@docker-compose stop app 2>/dev/null || true
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	air

run-prod: ## Run app in Docker container
	@docker-compose up -d app

dev-full: db-up dev ## Start database and run in development mode

killport: ## Kill process on port (usage: make killport PORT=8080)
	@echo "Killing process on port $(PORT)..."
	@lsof -ti:$(PORT) | xargs kill -9 2>/dev/null || echo "No process found on port $(PORT)"

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	swag init -g cmd/api/main.go --parseDependency --parseInternal --parseDepth 2 --output docs/swagger
# Test users management
user-test: ## Create 5 test users with access tokens (run on VPS)
	@echo "🔧 Creating test users in remote container..."
	@docker exec meeting-assistant-app ./create-test-users

user-clean: ## Delete all test users
	@echo "🧹 Deleting test users..."
	@docker exec meeting-assistant-postgres psql -U postgres -d meeting_assistant -c "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@test.local'); DELETE FROM users WHERE email LIKE '%@test.local'; SELECT 'Deleted test users';"