.PHONY: build test test-integration lint fmt sqlc dev api worker seed seed-demo web docker up down clean check helm-lint helm-template

BIN := bin/ticketowl
DATABASE_URL ?= postgres://ticketowl:ticketowl@localhost:5434/ticketowl?sslmode=disable

VERSION ?= 0.1.0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X github.com/wisbric/ticketowl/internal/version.Version=$(VERSION) \
           -X github.com/wisbric/ticketowl/internal/version.Commit=$(COMMIT)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/ticketowl

RACE := $(shell go env CGO_ENABLED 2>/dev/null | grep -q 1 && echo "-race" || echo "")

test:
	go test $(RACE) -count=1 ./...

test-integration:
	go test $(RACE) -count=1 -tags=integration ./...

lint:
	golangci-lint run ./...

fmt:
	goimports -w -local github.com/wisbric/ticketowl .
	gofmt -s -w .

sqlc:
	sqlc generate

dev: up api

api:
	go run ./cmd/ticketowl -mode api

worker:
	go run ./cmd/ticketowl -mode worker

seed:
	go run ./cmd/ticketowl -mode seed

seed-demo:
	go run ./cmd/ticketowl -mode seed-demo

web:
	cd web && npm run dev

docker:
	docker build -t ticketowl:dev .

up:
	docker compose up -d

down:
	docker compose down

clean:
	docker compose down -v
	rm -rf bin/ coverage.out internal/db/

check: lint test
	go vet ./...

helm-lint:
	helm lint charts/ticketowl/

helm-template:
	helm template ticketowl charts/ticketowl/
