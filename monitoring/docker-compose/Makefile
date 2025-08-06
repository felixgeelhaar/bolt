# Makefile for Bolt Monitoring Stack
# Comprehensive development and production management commands

.PHONY: help up down restart logs clean build test health monitor setup ssl cert
.DEFAULT_GOAL := help

# Variables
COMPOSE_FILE = docker-compose.yml
PROJECT_NAME = bolt-monitoring
SERVICES = bolt-app prometheus grafana jaeger elasticsearch kibana alertmanager fluentd
SSL_DIR = ./certs
SSL_CERT = $(SSL_DIR)/nginx.crt
SSL_KEY = $(SSL_DIR)/nginx.key

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
BLUE = \033[0;34m
PURPLE = \033[0;35m
CYAN = \033[0;36m
WHITE = \033[1;37m
NC = \033[0m # No Color

help: ## Show this help message
	@echo "$(CYAN)Bolt Logging Library - Monitoring Stack$(NC)"
	@echo "$(YELLOW)Available commands:$(NC)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ssl dirs ## Initial setup for development environment
	@echo "$(BLUE)Setting up Bolt monitoring environment...$(NC)"
	@mkdir -p logs data/postgres data/redis data/elasticsearch
	@chmod 755 logs data/postgres data/redis data/elasticsearch
	@echo "$(GREEN)Setup complete!$(NC)"

ssl: $(SSL_CERT) ## Generate SSL certificates for HTTPS

$(SSL_CERT): $(SSL_KEY)

$(SSL_KEY):
	@echo "$(BLUE)Generating SSL certificates...$(NC)"
	@mkdir -p $(SSL_DIR)
	@openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
		-keyout $(SSL_KEY) \
		-out $(SSL_CERT) \
		-subj "/C=US/ST=CA/L=San Francisco/O=Bolt Logging/OU=Development/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,DNS:bolt-monitoring.local,DNS:monitoring.bolt-monitoring.local,IP:127.0.0.1"
	@echo "$(GREEN)SSL certificates generated in $(SSL_DIR)$(NC)"

dirs: ## Create necessary directories
	@echo "$(BLUE)Creating directories...$(NC)"
	@mkdir -p logs/{bolt-app,nginx,fluentd}
	@mkdir -p data/{postgres,redis,elasticsearch,prometheus,grafana,alertmanager}
	@mkdir -p configs/{grafana/provisioning/{dashboards,datasources,notifiers},elasticsearch,kibana,postgres}
	@chmod -R 755 logs data configs
	@echo "$(GREEN)Directories created$(NC)"

up: setup ## Start all services
	@echo "$(BLUE)Starting Bolt monitoring stack...$(NC)"
	@docker-compose -p $(PROJECT_NAME) up -d
	@echo "$(GREEN)Stack started successfully!$(NC)"
	@echo "$(YELLOW)Available services:$(NC)"
	@echo "  - Bolt App:      https://localhost/ (Main application)"
	@echo "  - Prometheus:    https://localhost/prometheus/ (Metrics)"
	@echo "  - Grafana:       https://localhost/grafana/ (Dashboards)"
	@echo "  - Jaeger:        https://localhost/jaeger/ (Tracing)"
	@echo "  - AlertManager:  https://localhost/alertmanager/ (Alerts)"
	@echo "  - Kibana:        https://localhost/kibana/ (Logs)"
	@echo "  - Monitoring:    https://monitoring.bolt-monitoring.local/ (Dashboard hub)"

down: ## Stop all services
	@echo "$(RED)Stopping Bolt monitoring stack...$(NC)"
	@docker-compose -p $(PROJECT_NAME) down
	@echo "$(GREEN)Stack stopped$(NC)"

restart: down up ## Restart all services

logs: ## Show logs for all services
	@docker-compose -p $(PROJECT_NAME) logs -f

logs-app: ## Show logs for Bolt application only
	@docker-compose -p $(PROJECT_NAME) logs -f bolt-app

logs-prometheus: ## Show Prometheus logs
	@docker-compose -p $(PROJECT_NAME) logs -f prometheus

logs-grafana: ## Show Grafana logs
	@docker-compose -p $(PROJECT_NAME) logs -f grafana

logs-jaeger: ## Show Jaeger logs
	@docker-compose -p $(PROJECT_NAME) logs -f jaeger

ps: ## Show running containers status
	@docker-compose -p $(PROJECT_NAME) ps

health: ## Run health checks for all services
	@echo "$(BLUE)Running health checks...$(NC)"
	@docker-compose -p $(PROJECT_NAME) run --rm healthcheck
	@echo "$(GREEN)Health checks completed$(NC)"

monitor: ## Show real-time system monitoring
	@echo "$(CYAN)Real-time monitoring (Press Ctrl+C to stop)$(NC)"
	@watch -n 2 'docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" --no-stream | grep bolt'

build: ## Build custom images
	@echo "$(BLUE)Building Bolt application image...$(NC)"
	@cd ../../examples/cloud-native/docker-compose/app && docker build -t bolt-logging/app:v2.0.0 .
	@echo "$(GREEN)Build completed$(NC)"

test: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@./scripts/test-monitoring-stack.sh
	@echo "$(GREEN)Tests completed$(NC)"

clean: ## Clean up containers, volumes, and networks
	@echo "$(RED)Cleaning up Bolt monitoring stack...$(NC)"
	@docker-compose -p $(PROJECT_NAME) down -v --remove-orphans
	@docker system prune -f
	@echo "$(GREEN)Cleanup completed$(NC)"

clean-all: clean ## Complete cleanup including images
	@echo "$(RED)Removing all images and data...$(NC)"
	@docker-compose -p $(PROJECT_NAME) down -v --rmi all --remove-orphans
	@sudo rm -rf data/* logs/*
	@echo "$(GREEN)Complete cleanup finished$(NC)"

backup: ## Backup monitoring data
	@echo "$(BLUE)Creating backup...$(NC)"
	@mkdir -p backups/$(shell date +%Y%m%d-%H%M%S)
	@docker-compose -p $(PROJECT_NAME) exec -T prometheus tar czf - /prometheus | cat > backups/$(shell date +%Y%m%d-%H%M%S)/prometheus.tar.gz
	@docker-compose -p $(PROJECT_NAME) exec -T grafana tar czf - /var/lib/grafana | cat > backups/$(shell date +%Y%m%d-%H%M%S)/grafana.tar.gz
	@cp -r data backups/$(shell date +%Y%m%d-%H%M%S)/
	@echo "$(GREEN)Backup created in backups/$(shell date +%Y%m%d-%H%M%S)/$(NC)"

restore: ## Restore from backup (Usage: make restore BACKUP_DIR=20231201-120000)
	@if [ -z "$(BACKUP_DIR)" ]; then \
		echo "$(RED)Error: Please specify BACKUP_DIR. Usage: make restore BACKUP_DIR=20231201-120000$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Restoring from backup $(BACKUP_DIR)...$(NC)"
	@docker-compose -p $(PROJECT_NAME) down
	@cat backups/$(BACKUP_DIR)/prometheus.tar.gz | docker run --rm -i -v bolt-monitoring_prometheus_data:/prometheus alpine tar xzf - -C /
	@cat backups/$(BACKUP_DIR)/grafana.tar.gz | docker run --rm -i -v bolt-monitoring_grafana_data:/var/lib/grafana alpine tar xzf - -C /
	@cp -r backups/$(BACKUP_DIR)/data/* data/
	@docker-compose -p $(PROJECT_NAME) up -d
	@echo "$(GREEN)Restore completed$(NC)"

update: ## Update all service images
	@echo "$(BLUE)Updating service images...$(NC)"
	@docker-compose -p $(PROJECT_NAME) pull
	@docker-compose -p $(PROJECT_NAME) up -d --remove-orphans
	@echo "$(GREEN)Update completed$(NC)"

scale: ## Scale services (Usage: make scale SERVICE=bolt-app REPLICAS=3)
	@if [ -z "$(SERVICE)" ] || [ -z "$(REPLICAS)" ]; then \
		echo "$(RED)Error: Please specify SERVICE and REPLICAS. Usage: make scale SERVICE=bolt-app REPLICAS=3$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Scaling $(SERVICE) to $(REPLICAS) replicas...$(NC)"
	@docker-compose -p $(PROJECT_NAME) up -d --scale $(SERVICE)=$(REPLICAS)
	@echo "$(GREEN)Scaling completed$(NC)"

perf-test: ## Run performance tests against Bolt application
	@echo "$(BLUE)Running performance tests...$(NC)"
	@./scripts/performance-test.sh
	@echo "$(GREEN)Performance tests completed$(NC)"

stress-test: ## Run stress tests
	@echo "$(BLUE)Running stress tests...$(NC)"
	@./scripts/stress-test.sh
	@echo "$(GREEN)Stress tests completed$(NC)"

load-test: ## Run load tests
	@echo "$(BLUE)Running load tests...$(NC)"
	@docker run --rm -i --network bolt-monitoring_monitoring \
		loadimpact/k6:latest run - <scripts/load-test.js
	@echo "$(GREEN)Load tests completed$(NC)"

security-scan: ## Run security scans on containers
	@echo "$(BLUE)Running security scans...$(NC)"
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy image --exit-code 1 bolt-logging/app:v2.0.0
	@echo "$(GREEN)Security scans completed$(NC)"

benchmark: ## Run Bolt performance benchmarks
	@echo "$(BLUE)Running Bolt benchmarks...$(NC)"
	@docker-compose -p $(PROJECT_NAME) exec bolt-app go test -bench=. -benchmem
	@echo "$(GREEN)Benchmarks completed$(NC)"

config-validate: ## Validate configuration files
	@echo "$(BLUE)Validating configurations...$(NC)"
	@docker run --rm -v $(PWD)/configs/prometheus:/prometheus prom/prometheus:v2.47.0 \
		promtool check config /prometheus/prometheus.yml
	@docker run --rm -v $(PWD)/configs/alertmanager:/alertmanager prom/alertmanager:v0.26.0 \
		amtool check-config /alertmanager/alertmanager.yml
	@echo "$(GREEN)Configuration validation completed$(NC)"

install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@if ! command -v docker-compose >/dev/null 2>&1; then \
		echo "Installing docker-compose..."; \
		curl -L "https://github.com/docker/compose/releases/download/v2.23.0/docker-compose-$$(uname -s)-$$(uname -m)" -o /usr/local/bin/docker-compose; \
		chmod +x /usr/local/bin/docker-compose; \
	fi
	@if ! command -v k6 >/dev/null 2>&1; then \
		echo "Installing k6 for load testing..."; \
		curl -s https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz | tar xvz -C /tmp; \
		sudo mv /tmp/k6-v0.47.0-linux-amd64/k6 /usr/local/bin/; \
	fi
	@echo "$(GREEN)Development tools installed$(NC)"

dashboard: ## Open monitoring dashboard in browser
	@echo "$(BLUE)Opening monitoring dashboard...$(NC)"
	@if command -v xdg-open >/dev/null 2>&1; then \
		xdg-open https://localhost/grafana/; \
	elif command -v open >/dev/null 2>&1; then \
		open https://localhost/grafana/; \
	else \
		echo "$(YELLOW)Please open https://localhost/grafana/ in your browser$(NC)"; \
	fi

shell: ## Open shell in Bolt application container
	@docker-compose -p $(PROJECT_NAME) exec bolt-app /bin/sh

shell-prometheus: ## Open shell in Prometheus container
	@docker-compose -p $(PROJECT_NAME) exec prometheus /bin/sh

shell-grafana: ## Open shell in Grafana container
	@docker-compose -p $(PROJECT_NAME) exec grafana /bin/bash

dev: setup up ## Start development environment
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo "$(YELLOW)Quick start:$(NC)"
	@echo "  1. Visit https://localhost/grafana/ (admin/bolt-admin-2024)"
	@echo "  2. Check application metrics: https://localhost/metrics"
	@echo "  3. View traces: https://localhost/jaeger/"
	@echo "  4. Monitor alerts: https://localhost/alertmanager/"
	@echo "  5. Search logs: https://localhost/kibana/"

prod: ## Prepare for production deployment
	@echo "$(BLUE)Preparing production configuration...$(NC)"
	@./scripts/prod-setup.sh
	@echo "$(GREEN)Production setup completed$(NC)"

# Quick development commands
quick-restart: ## Quick restart of main services
	@docker-compose -p $(PROJECT_NAME) restart bolt-app prometheus grafana

quick-logs: ## Quick view of important logs
	@docker-compose -p $(PROJECT_NAME) logs --tail=50 bolt-app prometheus grafana alertmanager

quick-check: ## Quick health check
	@echo "$(BLUE)Quick health check...$(NC)"
	@curl -s https://localhost/health || echo "$(RED)Application not responding$(NC)"
	@curl -s http://localhost:9000/-/healthy || echo "$(RED)Prometheus not responding$(NC)"
	@curl -s http://localhost:3000/api/health || echo "$(RED)Grafana not responding$(NC)"
	@echo "$(GREEN)Quick check completed$(NC)"