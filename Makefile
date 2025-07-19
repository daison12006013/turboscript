.DEFAULT_GOAL := help

# Development commands
up: ## Start everything (app + database) in development mode
	docker-compose -f docker-compose.dev.yml up -d

down: ## Stop everything
	docker-compose -f docker-compose.dev.yml down

restart: ## Restart everything
	docker-compose -f docker-compose.dev.yml restart

restart-clean: ## Restart everything with clean state (fixes Kafka restart issues)
	@echo "🔄 Performing clean restart to fix potential Kafka issues..."
	docker-compose -f docker-compose.dev.yml down
	@echo "⏳ Waiting for services to fully stop..."
	sleep 3
	@echo "🚀 Starting services..."
	docker-compose -f docker-compose.dev.yml up -d

logs: ## Show logs
	docker-compose -f docker-compose.dev.yml logs -f

rebuild: ## Rebuild and restart the app
	docker-compose -f docker-compose.dev.yml build app-dev
	docker-compose -f docker-compose.dev.yml up -d

# Database only
db: ## Start only the database
	docker-compose -f docker-compose.dev.yml up -d postgres

db-shell: ## Connect to database
	docker-compose -f docker-compose.dev.yml exec postgres psql -U turboscript_user -d turboscript

# Local development
build: ## Build the Go app locally
	@echo "🏗️  Building frontend..."
	npx tsx scripts/build-frontend.ts
	@echo "🏗️  Building Go app..."
	go build -o .bin/turboscript .

build-dist: ## Build for distribution (compile TS to JS and create dist folder)
	@echo "🔨 Building TurboScript for distribution..."
	@echo "📦 Installing Node.js dependencies..."
	npm ci
	@echo "🏗️  Building React frontend..."
	npx tsx scripts/build-frontend.ts
	@echo "🔧 Compiling TypeScript files to JavaScript..."
	npm run build:prod
	@echo "🏗️  Building Go binary..."
	go build -o dist/turboscript .
	@echo "📋 Copying configuration files..."
	cp turboscript.yml dist/
	@echo "📁 Copying documentation files..."
	cp README.md CONTRIBUTING.md LICENSE SECURITY.md dist/
	@echo "📦 Copying React frontend assets..."
	mkdir -p dist/app/hybrid
	cp -r app/hybrid/assets dist/app/hybrid/
	cp app/hybrid/App.html dist/app/hybrid/
	@echo "⚙️  Modifying configuration for production..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		sed -i '' 's/debug: true/debug: false/g' dist/turboscript.yml; \
		sed -i '' 's/monitoring: true/monitoring: false/g' dist/turboscript.yml; \
		sed -i '' 's/prefer_ts: true/prefer_js: true/g' dist/turboscript.yml; \
		echo '' >> dist/turboscript.yml; \
		echo '# Production build uses compiled JavaScript files' >> dist/turboscript.yml; \
		echo 'prefer_js: true' >> dist/turboscript.yml; \
	else \
		sed -i 's/debug: true/debug: false/g' dist/turboscript.yml; \
		sed -i 's/monitoring: true/monitoring: false/g' dist/turboscript.yml; \
		sed -i 's/prefer_ts: true/prefer_js: true/g' dist/turboscript.yml; \
		echo '' >> dist/turboscript.yml; \
		echo '# Production build uses compiled JavaScript files' >> dist/turboscript.yml; \
		echo 'prefer_js: true' >> dist/turboscript.yml; \
	fi
	@echo "🏗️  Building Go binary for Linux..."
	GOOS=linux GOARCH=amd64 go build -o dist/turboscript-linux .
	@echo "✅ Build complete! Check the dist/ folder"
	@echo "📝 Creating runner script..."
	echo '#!/bin/bash' > dist/runner.sh
	echo '# Configure database connections in turboscript.yml' >> dist/runner.sh
	echo './turboscript' >> dist/runner.sh
	chmod +x dist/runner.sh

run: ## Run the app locally (configure database in turboscript.yml)
	go run .

clean: ## Remove all Docker containers and volumes
	docker-compose -f docker-compose.dev.yml down -v

clean-dist: ## Remove dist folder
	rm -rf dist/

# Production commands
build-docker: ## Build production Docker image
	docker build -t turboscript:latest .

download:
	curl -sSL \
	    -o golangci-lint.tar.gz \
	    https://github.com/golangci/golangci-lint/releases/download/v2.2.1/golangci-lint-2.2.1-darwin-amd64.tar.gz
	tar -xvf golangci-lint.tar.gz
	sudo mv golangci-lint-*/golangci-lint .
	rm -rf golangci-lint-*/ golangci-lint.tar.gz

golangci:
	@echo "Running golangci-lint..."
	./golangci-lint run

find-fail: ## Find and display only failed tests from all test suites
	@echo "🔍 Searching for failed tests..."
	@echo "📋 Unit test failures:"
	@make test 2>&1 | grep "FAIL" || echo "  ✅ No unit test failures found"
	@echo "📋 Integration test failures:"
	@make test-integration 2>&1 | grep "FAIL" || echo "  ✅ No integration test failures found"
	@echo "📋 E2E test failures:"
	@make test-e2e 2>&1 | grep "FAIL" || echo "  ✅ No E2E test failures found"
	@echo "📋 GolangCI-Lint failures:"
	@./golangci-lint run 2>&1 | grep "FAIL" || echo "  ✅ No GolangCI-Lint failures found"
	@echo "📋 E2E benchmark:"
	@make test-e2e-bench 2>&1 | grep -E "(BenchmarkE2EEndpoints.*ns/op|goos:|goarch:|pkg:|cpu:)" || echo "  ⚠️  No benchmark results found"

# Testing commands
test: ## Run unit tests
	go test -v -race ./...

cov: ## Run tests with coverage analysis and generate HTML report
	@echo "🧪 Running tests with coverage analysis..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "📈 Coverage summary:"
	go tool cover -func=coverage.out | tail -1
	@echo "🌐 HTML coverage report generated: coverage.html"
	@echo "💡 Tip: Open coverage.html in your browser to see detailed coverage"

test-integration: ## Run integration tests (requires running server)
	INTEGRATION_TEST=true go test -v -run TestIntegrationEndpoints ./...

test-e2e: ## Run end-to-end tests (requires running server)
	E2E_TEST=true go test -v -run TestE2EEndpoints ./...

test-e2e-auto: ## Run end-to-end tests with automatic server management
	./scripts/e2e-helper.sh test

test-e2e-bench: ## Run end-to-end performance benchmarks (requires running server)
	E2E_TEST=true go test -v -bench=BenchmarkE2EEndpoints -run=^$$ -benchmem ./internal/tests/

test-e2e-bench-auto: ## Run end-to-end benchmarks with automatic server management
	./scripts/e2e-helper.sh bench

test-e2e-full: ## Run complete e2e test suite with server lifecycle management
	./scripts/e2e-helper.sh full

test-all: ## Run all tests (unit, integration, e2e)
	@echo "🧪 Running unit tests..."
	go test -count=1 -v -race  ./...
	@echo "🔗 Running integration tests..."
	INTEGRATION_TEST=true go test -count=1 -v -run TestIntegrationEndpoints ./...
	@echo "🎯 Running end-to-end tests..."
	E2E_TEST=true go test -count=1 -v -run TestE2EEndpoints ./...

# Postman contract testing
postman-install: ## Install Newman for Postman contract testing
	./scripts/run-postman-contract.sh --install

postman-test: ## Run Postman API contract tests (requires running server)
	./scripts/run-postman-contract.sh

postman-test-auto: ## Run Postman contract tests with automatic server management
	@echo "🚀 Starting server for Postman contract testing..."
	make up
	@sleep 5
	@echo "📄 Running Postman contract tests..."
	./scripts/run-postman-contract.sh || (make down && exit 1)
	@echo "🛑 Stopping server..."
	make down
	@echo "✅ Postman contract testing completed"

test-full-contract: ## Run complete test suite including Postman contract
	@echo "🧪 Running complete test suite with Postman contract..."
	@echo "📋 Step 1: Unit tests"
	go test -v -race ./...
	@echo "📋 Step 2: Starting server for integration tests"
	make up
	@sleep 5
	@echo "📋 Step 3: Integration tests"
	INTEGRATION_TEST=true go test -v -run TestIntegrationEndpoints ./... || (make down && exit 1)
	@echo "📋 Step 4: E2E tests"
	E2E_TEST=true go test -v -run TestE2EEndpoints ./... || (make down && exit 1)
	@echo "📋 Step 5: Postman contract tests"
	./scripts/run-postman-contract.sh || (make down && exit 1)
	@echo "📋 Step 6: Stopping server"
	make down
	@echo "🎉 Complete test suite with contract testing passed!"

# Security commands
security: ## Run comprehensive security scan
	@echo "🔒 Running security analysis..."
	./scripts/security-scan.sh

security-gosec: ## Run only Gosec security scanner
	@echo "🔍 Running Gosec security scanner..."
	./scripts/security-scan.sh --gosec-only

security-deps: ## Check for vulnerable dependencies
	@echo "📦 Checking dependencies for vulnerabilities..."
	./scripts/security-scan.sh --nancy-only

# CI/CD and Self-Hosted Runner commands
ci-local: ## Run complete CI pipeline locally
	@echo "🚀 Running complete CI pipeline locally..."
	./scripts/ci-local.sh

ci-local-fast: ## Run CI pipeline locally (skip performance and security)
	@echo "🚀 Running fast CI pipeline locally..."
	./scripts/ci-local.sh --skip-performance --skip-security

ci-local-essential: ## Run only essential CI tests (lint, test, build)
	@echo "🚀 Running essential CI tests locally..."
	./scripts/ci-local.sh --skip-e2e --skip-postman --skip-security --skip-docker

ci-act: ## Run GitHub Actions locally using act
	@echo "🎭 Running GitHub Actions locally using act..."
	./scripts/ci-act.sh

ci-act-list: ## List available GitHub Actions jobs
	@echo "📋 Listing available GitHub Actions jobs..."
	./scripts/ci-act.sh --list

ci-act-dry: ## Dry run GitHub Actions locally
	@echo "🔍 Dry run GitHub Actions locally..."
	./scripts/ci-act.sh --dry-run

ci-act-install: ## Install act (GitHub Actions local runner)
	@echo "📦 Installing act..."
	@if command -v brew >/dev/null 2>&1; then \
		brew install act; \
	elif command -v curl >/dev/null 2>&1; then \
		curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash; \
	else \
		echo "❌ Please install act manually from: https://github.com/nektos/act/releases"; \
		exit 1; \
	fi
	@echo "✅ Act installed successfully"

ci-setup: ## Setup self-hosted GitHub Actions runner
	@echo "🔧 Setting up self-hosted GitHub Actions runner..."
	./scripts/setup-ci.sh

ci-setup-config: ## Configure self-hosted runner environment only
	@echo "⚙️  Configuring self-hosted runner environment..."
	./scripts/setup-ci.sh --config-only

ci-down: ## Stop self-hosted GitHub Actions runner
	@echo "🛑 Stopping self-hosted GitHub Actions runner..."
	cd .github/runner && docker-compose down

ci-restart: ## Restart self-hosted GitHub Actions runner
	@echo "🔄 Restarting self-hosted GitHub Actions runner..."
	cd .github/runner && docker-compose restart

ci-logs: ## View GitHub Actions runner logs
	@echo "📋 Viewing GitHub Actions runner logs..."
	cd .github/runner && docker-compose logs -f github-runner

ci-status: ## Check GitHub Actions runner status
	@echo "📊 Checking GitHub Actions runner status..."
	cd .github/runner && docker-compose ps

ci-shell: ## Connect to GitHub Actions runner container
	@echo "🐚 Connecting to GitHub Actions runner container..."
	cd .github/runner && docker-compose exec github-runner bash

ci-rebuild: ## Rebuild GitHub Actions runner image
	@echo "🔨 Rebuilding GitHub Actions runner image..."
	cd .github/runner && docker-compose build --no-cache

ci-clean: ## Remove GitHub Actions runner containers and volumes
	@echo "🧹 Cleaning up GitHub Actions runner..."
	cd .github/runner && docker-compose down -v
	docker image prune -f

ci-config: ## Show GitHub Actions runner configuration help
	@echo "📖 GitHub Actions Runner Configuration Help"
	@echo "============================================="
	@echo ""
	@echo "1. Copy the environment template:"
	@echo "   cp .github/runner/.env.example .github/runner/.env"
	@echo ""
	@echo "2. Edit .github/runner/.env with your settings:"
	@echo "   - GITHUB_TOKEN: Personal access token with 'repo' scope"
	@echo "   - GITHUB_REPOSITORY: your-username/turboscript"
	@echo ""
	@echo "3. Start the runner:"
	@echo "   make ci-setup"
	@echo ""
	@echo "4. Verify runner is registered:"
	@echo "   Check your repository Settings > Actions > Runners"
	@echo ""
	@echo "For detailed setup instructions, see: .github/runner/README.md"

help: ## Show available commands
	@echo 'Usage: make [command]'
	@echo ''
	@echo 'Commands:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

%:
	@:
