# Bolt Examples Makefile
# Comprehensive build and deployment automation for all enterprise examples

.PHONY: help build test clean run docker k8s-deploy k8s-clean lint format deps

# Variables
GO_VERSION := 1.23
DOCKER_REGISTRY := bolt-examples
KUBERNETES_NAMESPACE := bolt-examples
PROJECT_ROOT := $(shell pwd)/..

# Default target
help: ## Show this help message
	@echo "Bolt Enterprise Integration Examples"
	@echo "===================================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependencies
deps: ## Install all dependencies
	@echo "Installing dependencies for all examples..."
	@cd microservices/http-middleware && go mod tidy
	@cd microservices/grpc-interceptors && go mod tidy
	@cd observability/opentelemetry && go mod tidy
	@cd observability/prometheus && go mod tidy
	@cd security/audit-logging && go mod tidy
	@cd security/pii-masking && go mod tidy
	@cd high-availability/load-balancer && go mod tidy
	@echo "✓ All dependencies installed"

# Build targets
build: ## Build all examples
	@echo "Building all examples..."
	@$(MAKE) build-microservices
	@$(MAKE) build-observability
	@$(MAKE) build-security
	@$(MAKE) build-ha
	@echo "✓ All examples built successfully"

build-microservices: ## Build microservices examples
	@echo "Building microservices examples..."
	@cd microservices/http-middleware && go build -o bin/http-middleware .
	@cd microservices/grpc-interceptors && go build -o bin/grpc-interceptors .
	@echo "✓ Microservices examples built"

build-observability: ## Build observability examples
	@echo "Building observability examples..."
	@cd observability/opentelemetry && go build -o bin/opentelemetry .
	@cd observability/prometheus && go build -o bin/prometheus .
	@echo "✓ Observability examples built"

build-security: ## Build security examples
	@echo "Building security examples..."
	@cd security/audit-logging && go build -o bin/audit-logging .
	@cd security/pii-masking && go build -o bin/pii-masking .
	@echo "✓ Security examples built"

build-ha: ## Build high-availability examples
	@echo "Building high-availability examples..."
	@cd high-availability/load-balancer && go build -o bin/load-balancer .
	@echo "✓ High-availability examples built"

# Test targets
test: ## Run all tests
	@echo "Running tests for all examples..."
	@$(MAKE) test-microservices
	@$(MAKE) test-observability
	@$(MAKE) test-security
	@$(MAKE) test-ha
	@echo "✓ All tests completed"

test-microservices: ## Test microservices examples
	@echo "Testing microservices examples..."
	@cd microservices/http-middleware && go test -v -race ./...
	@cd microservices/grpc-interceptors && go test -v -race ./...

test-observability: ## Test observability examples
	@echo "Testing observability examples..."
	@cd observability/opentelemetry && go test -v -race ./...
	@cd observability/prometheus && go test -v -race ./...

test-security: ## Test security examples
	@echo "Testing security examples..."
	@cd security/audit-logging && go test -v -race ./...
	@cd security/pii-masking && go test -v -race ./...

test-ha: ## Test high-availability examples
	@echo "Testing high-availability examples..."
	@cd high-availability/load-balancer && go test -v -race ./...

# Code quality targets
lint: ## Run linters on all examples
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@cd microservices/http-middleware && golangci-lint run
	@cd microservices/grpc-interceptors && golangci-lint run
	@cd observability/opentelemetry && golangci-lint run
	@cd observability/prometheus && golangci-lint run
	@cd security/audit-logging && golangci-lint run
	@cd security/pii-masking && golangci-lint run
	@cd high-availability/load-balancer && golangci-lint run
	@echo "✓ Linting completed"

format: ## Format all Go code
	@echo "Formatting Go code..."
	@find . -name "*.go" -exec gofmt -s -w {} \;
	@find . -name "*.go" -exec goimports -w {} \;
	@echo "✓ Code formatted"

# Runtime targets
run-http-middleware: ## Run HTTP middleware example
	@echo "Starting HTTP middleware example..."
	@cd microservices/http-middleware && go run .

run-grpc-interceptors: ## Run gRPC interceptors example (server)
	@echo "Starting gRPC interceptors example..."
	@cd microservices/grpc-interceptors && go run .

run-grpc-client: ## Run gRPC client example
	@echo "Starting gRPC client example..."
	@cd microservices/grpc-interceptors && go run . client

run-opentelemetry: ## Run OpenTelemetry example
	@echo "Starting OpenTelemetry example..."
	@cd observability/opentelemetry && go run .

run-prometheus: ## Run Prometheus example
	@echo "Starting Prometheus example..."
	@cd observability/prometheus && go run .

run-audit-logging: ## Run audit logging example
	@echo "Starting audit logging example..."
	@cd security/audit-logging && go run .

run-pii-masking: ## Run PII masking example
	@echo "Starting PII masking example..."
	@cd security/pii-masking && go run .

run-load-balancer: ## Run load balancer example
	@echo "Starting load balancer example..."
	@cd high-availability/load-balancer && go run .

run-backend: ## Run backend server (specify PORT=8081,8082,8083)
	@echo "Starting backend server on port $(PORT)..."
	@cd high-availability/load-balancer && go run . backend $(PORT)

# Docker targets
docker-build: ## Build all Docker images
	@echo "Building Docker images..."
	@$(MAKE) docker-build-microservices
	@$(MAKE) docker-build-observability
	@$(MAKE) docker-build-security
	@$(MAKE) docker-build-ha
	@echo "✓ All Docker images built"

docker-build-microservices: ## Build microservices Docker images
	@echo "Building microservices Docker images..."
	@cd microservices/http-middleware && docker build -t $(DOCKER_REGISTRY)/http-middleware:latest .
	@cd microservices/grpc-interceptors && docker build -t $(DOCKER_REGISTRY)/grpc-interceptors:latest .

docker-build-observability: ## Build observability Docker images
	@echo "Building observability Docker images..."
	@cd observability/opentelemetry && docker build -t $(DOCKER_REGISTRY)/opentelemetry:latest .
	@cd observability/prometheus && docker build -t $(DOCKER_REGISTRY)/prometheus:latest .

docker-build-security: ## Build security Docker images
	@echo "Building security Docker images..."
	@cd security/audit-logging && docker build -t $(DOCKER_REGISTRY)/audit-logging:latest .
	@cd security/pii-masking && docker build -t $(DOCKER_REGISTRY)/pii-masking:latest .

docker-build-ha: ## Build high-availability Docker images
	@echo "Building high-availability Docker images..."
	@cd high-availability/load-balancer && docker build -t $(DOCKER_REGISTRY)/load-balancer:latest .

# Docker Compose targets
compose-up: ## Start all services with Docker Compose
	@echo "Starting services with Docker Compose..."
	@cd cloud-native/docker-compose && docker-compose up -d
	@echo "✓ Services started. Access at:"
	@echo "  Application: http://localhost:8080"
	@echo "  Grafana: http://localhost:3000 (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Jaeger: http://localhost:16686"

compose-down: ## Stop all services
	@echo "Stopping Docker Compose services..."
	@cd cloud-native/docker-compose && docker-compose down
	@echo "✓ Services stopped"

compose-logs: ## Show logs from all services
	@cd cloud-native/docker-compose && docker-compose logs -f

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	@kubectl create namespace $(KUBERNETES_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@cd cloud-native/kubernetes && kubectl apply -f manifests/ -n $(KUBERNETES_NAMESPACE)
	@echo "✓ Deployed to Kubernetes namespace: $(KUBERNETES_NAMESPACE)"

k8s-status: ## Check Kubernetes deployment status
	@echo "Kubernetes deployment status:"
	@kubectl get all -n $(KUBERNETES_NAMESPACE)

k8s-logs: ## Show Kubernetes logs
	@kubectl logs -f deployment/bolt-demo-app -n $(KUBERNETES_NAMESPACE)

k8s-clean: ## Clean up Kubernetes resources
	@echo "Cleaning up Kubernetes resources..."
	@cd cloud-native/kubernetes && kubectl delete -f manifests/ -n $(KUBERNETES_NAMESPACE) --ignore-not-found=true
	@kubectl delete namespace $(KUBERNETES_NAMESPACE) --ignore-not-found=true
	@echo "✓ Kubernetes resources cleaned up"

# Development targets
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	@$(MAKE) deps
	@$(MAKE) format
	@$(MAKE) lint
	@$(MAKE) build
	@$(MAKE) test
	@echo "✓ Development environment ready"

demo-microservices: ## Run microservices demo
	@echo "Starting microservices demo..."
	@echo "Starting HTTP middleware server..."
	@cd microservices/http-middleware && go run . &
	@sleep 2
	@echo "Testing HTTP middleware endpoints..."
	@curl -H "X-Correlation-ID: demo-123" http://localhost:8080/users
	@echo ""
	@curl -X POST http://localhost:8080/users
	@echo ""
	@curl http://localhost:8080/health
	@echo ""
	@pkill -f "http-middleware"

demo-observability: ## Run observability demo
	@echo "Starting observability demo..."
	@cd observability/prometheus && go run . &
	@sleep 2
	@echo "Testing Prometheus metrics endpoints..."
	@curl http://localhost:8080/
	@echo ""
	@curl http://localhost:8080/users
	@echo ""
	@curl http://localhost:8080/metrics | head -20
	@echo ""
	@pkill -f "prometheus"

demo-security: ## Run security demo
	@echo "Starting security demo..."
	@cd security/audit-logging && go run . &
	@sleep 2
	@echo "Testing audit logging endpoints..."
	@curl -X POST -H "X-User-ID: user123" http://localhost:8080/auth/login -d "email=john@example.com&password=secret"
	@echo ""
	@curl -H "X-User-ID: user123" "http://localhost:8080/users/data?user_id=user123"
	@echo ""
	@pkill -f "audit-logging"

demo-ha: ## Run high-availability demo
	@echo "Starting high-availability demo..."
	@echo "Starting backend servers..."
	@cd high-availability/load-balancer && go run . backend 8081 &
	@cd high-availability/load-balancer && go run . backend 8082 &
	@cd high-availability/load-balancer && go run . backend 8083 &
	@sleep 2
	@echo "Starting load balancer..."
	@cd high-availability/load-balancer && go run . &
	@sleep 3
	@echo "Testing load balancer..."
	@curl http://localhost:8080/
	@echo ""
	@curl http://localhost:8080/lb/stats
	@echo ""
	@curl http://localhost:8080/lb/health
	@echo ""
	@pkill -f "load-balancer"

# Performance testing
perf-test: ## Run performance tests
	@echo "Running performance tests..."
	@command -v hey >/dev/null 2>&1 || { echo "Installing hey..."; go install github.com/rakyll/hey@latest; }
	@cd microservices/http-middleware && go run . &
	@sleep 3
	@echo "Performance testing HTTP middleware..."
	@hey -n 1000 -c 10 http://localhost:8080/users
	@pkill -f "http-middleware"

# Benchmarking
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@cd $(PROJECT_ROOT) && go test -bench=. -benchmem ./...
	@echo "Example-specific benchmarks:"
	@find . -name "*_test.go" -exec dirname {} \; | sort -u | xargs -I {} sh -c 'cd {} && if [ -f "*_test.go" ]; then go test -bench=. -benchmem; fi'

# Monitoring setup
monitoring-setup: ## Set up monitoring stack
	@echo "Setting up monitoring stack..."
	@cd cloud-native/docker-compose && docker-compose -f docker-compose.yml up -d prometheus grafana jaeger elasticsearch kibana
	@echo "✓ Monitoring stack started:"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana: http://localhost:3000"
	@echo "  Jaeger: http://localhost:16686"
	@echo "  Kibana: http://localhost:5601"

# Cleanup
clean: ## Clean up build artifacts
	@echo "Cleaning up build artifacts..."
	@find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null || true
	@find . -name "*.log" -delete 2>/dev/null || true
	@find . -name "coverage.out" -delete 2>/dev/null || true
	@docker system prune -f 2>/dev/null || true
	@echo "✓ Cleanup completed"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@command -v godoc >/dev/null 2>&1 || { echo "Installing godoc..."; go install golang.org/x/tools/cmd/godoc@latest; }
	@echo "Starting local documentation server..."
	@echo "Documentation available at: http://localhost:6060/pkg/github.com/felixgeelhaar/bolt/"
	@godoc -http=:6060

# CI/CD helpers
ci-setup: ## Set up CI environment
	@echo "Setting up CI environment..."
	@go version
	@$(MAKE) deps
	@$(MAKE) format
	@$(MAKE) lint
	@$(MAKE) build
	@$(MAKE) test
	@echo "✓ CI setup completed"

release-check: ## Check if ready for release
	@echo "Performing release checks..."
	@$(MAKE) ci-setup
	@$(MAKE) perf-test
	@echo "✓ Release checks completed - ready for release"

# Utility targets
version: ## Show version information
	@echo "Bolt Examples Version Information"
	@echo "================================"
	@echo "Go version: $(shell go version)"
	@echo "Docker version: $(shell docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Kubernetes client: $(shell kubectl version --client --short 2>/dev/null || echo 'Not installed')"
	@echo "Project root: $(PROJECT_ROOT)"
	@echo "Examples directory: $(shell pwd)"

status: ## Show current status
	@echo "Bolt Examples Status"
	@echo "==================="
	@echo "Running processes:"
	@ps aux | grep -E "(http-middleware|grpc-interceptors|opentelemetry|prometheus|audit-logging|pii-masking|load-balancer)" | grep -v grep || echo "No examples running"
	@echo ""
	@echo "Docker containers:"
	@docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Docker not available"
	@echo ""
	@echo "Kubernetes pods:"
	@kubectl get pods -n $(KUBERNETES_NAMESPACE) 2>/dev/null || echo "No Kubernetes pods found"

# All-in-one targets
demo-all: ## Run all demos sequentially
	@echo "Running all demos..."
	@$(MAKE) demo-microservices
	@sleep 2
	@$(MAKE) demo-observability
	@sleep 2
	@$(MAKE) demo-security
	@sleep 2
	@$(MAKE) demo-ha
	@echo "✓ All demos completed"

full-stack: ## Deploy full observability stack
	@echo "Deploying full observability stack..."
	@$(MAKE) monitoring-setup
	@sleep 10
	@$(MAKE) run-opentelemetry &
	@$(MAKE) run-prometheus &
	@sleep 5
	@echo "✓ Full stack deployed and running"
	@echo "Access points:"
	@echo "  Application (OpenTelemetry): http://localhost:8080"
	@echo "  Application (Prometheus): http://localhost:8081"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana: http://localhost:3000"
	@echo "  Jaeger: http://localhost:16686"