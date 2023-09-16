## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: run the cmd/api application
.PHONY: run
run:
	@go run ./cmd/api -db-dsn $(GREENLIGHT_DB_DSN)

## test: run tests
.PHONY: test
test:
	go test ./...

## lint: run linter
.PHONY: lint
lint:
	golangci-lint run

## migration name=$1: create a new DB migration
.PHONY: migration
migration:
	migrate create -seq -ext=.sql -dir=./migrations $(name)

## migrate: apply DB migrations
.PHONY: migrate
migrate:
	migrate -path=./migrations -database=$(GREENLIGHT_DB_DSN) up

## psql: connect to the db using psql
.PHONY: psql
psql:
	psql $(GREENLIGHT_DB_DSN)
