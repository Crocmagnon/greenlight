## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: run the cmd/api application
.PHONY: run
run:
	@go run ./cmd/api -db-dsn $(GREENLIGHT_DB_DSN)

## build: build the cmd/api application
.PHONY: build
build:
	go build -ldflags='-s -w' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o=./bin/api-linux-amd64 ./cmd/api

## test: run tests
.PHONY: test
test:
	go test ./...

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

## lint: run linter
.PHONY: lint
lint:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	golangci-lint run
	@echo 'Running tests...'
	go test -race -vet=off ./...
