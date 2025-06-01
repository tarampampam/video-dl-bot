#!/usr/bin/make

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"

.DEFAULT_GOAL : help
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

shell: ## Start shell
	docker compose run $(DC_RUN_ARGS) app bash

lint: ## Run linters
	docker compose run $(DC_RUN_ARGS) app golangci-lint run

gen: ## Generate code
	docker compose run $(DC_RUN_ARGS) app go generate ./...

up: ## Start the application in watch mode
	echo "not implemented yet"

down: ## Stop the application
	docker compose down --remove-orphans
