.PHONY: up down test test-cover tidy fmt fmt-check vet lint build check check-quick install-tools install-hooks setup

# ============================================================================
# Development Environment
# ============================================================================

up:
	docker compose up -d

down:
	docker compose down

# ============================================================================
# Code Quality
# ============================================================================

# Format all Go files
fmt:
	@echo "🔧 Formatting code..."
	@gofmt -w .
	@echo "✅ Formatting complete"

# Check formatting without modifying files
fmt-check:
	@echo "🔍 Checking formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ The following files need formatting:"; \
		gofmt -l .; \
		echo ""; \
		echo "Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@echo "✅ All files properly formatted"

# Run go vet
vet:
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Vet passed"

# Run staticcheck (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)
lint:
	@echo "🔍 Running golangci-lint..."
	@$$(go env GOPATH)/bin/golangci-lint run ./... || (echo "💡 Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@echo "✅ golangci-lint passed"

# Build all packages (verifies compilation)
build:
	@echo "🔨 Building all packages..."
	@go build ./...
	@echo "✅ Build successful"

# ============================================================================
# Testing
# ============================================================================

test:
	@echo "🧪 Running tests..."
	@go list ./... | grep -E "pkg|templates|services" | xargs -r go test -race
	@echo "✅ All tests passed"

test-cover:
	@echo "🧪 Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./pkg/...
	@go tool cover -func=coverage.out
	@echo "📊 Detailed HTML report: go tool cover -html=coverage.out"

# ============================================================================
# Quality Gates (Run before pushing)
# ============================================================================

# Full quality check - run this before pushing
check: fmt-check vet lint build test
	@echo ""
	@echo "🎉 All quality gates passed! Safe to push."

# Quick check - format, vet, and build only (skip tests for speed)
check-quick: fmt-check vet build
	@echo ""
	@echo "⚡ Quick check passed! (tests skipped)"

# ============================================================================
# Dependencies
# ============================================================================

tidy:
	go mod tidy
	@for dir in ./pkg/*; do \
		[ -d "$$dir" ] && (cd "$$dir" && [ -f go.mod ] && go mod tidy) || true; \
	done
	@for dir in ./services/*; do \
		[ -d "$$dir" ] && (cd "$$dir" && [ -f go.mod ] && go mod tidy) || true; \
	done
	@for dir in ./templates/*; do \
		[ -d "$$dir" ] && (cd "$$dir" && [ -f go.mod ] && go mod tidy) || true; \
	done
	@for dir in ./verification/*; do \
		[ -d "$$dir" ] && (cd "$$dir" && [ -f go.mod ] && go mod tidy) || true; \
	done

# Install development tools
install-tools:
	@echo "📦 Installing development tools..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ Tools installed"

# Install git hooks
install-hooks:
	@echo "🔗 Installing git hooks..."
	@cp .github/hooks/pre-push .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✅ Pre-push hook installed"

# Full development setup
setup: install-tools install-hooks
	@echo "🎉 Development environment ready!"
