test:
	go test ./...

lint:
	golangci-lint run

migration:
	migrate create -seq -ext=.sql -dir=./migrations $(name)

migrate:
	migrate -path=./migrations -database=$(GREENLIGHT_DB_DSN) up

dbconnect:
	psql $(GREENLIGHT_DB_DSN)
