.PHONY: all install build test lint clean dev-backend dev-frontend dev-cli

BACKEND_BIN := backend/bin/cloudcontrol
CLI_BIN     := cli/bin/cloudctl

all: build

install: install-backend install-cli install-frontend

install-backend:
	cd backend && go mod download

install-cli:
	cd cli && go mod download

install-frontend:
	cd frontend && npm install

build: build-backend build-cli build-frontend

build-backend:
	cd backend && go build -ldflags="-s -w" -o bin/cloudcontrol ./cmd/server

build-cli:
	cd cli && go build -ldflags="-s -w" -o bin/cloudctl ./cmd/cloudctl && \
	  sudo cp bin/cloudctl /usr/local/bin/cloudctl 2>/dev/null || cp bin/cloudctl ~/.local/bin/cloudctl

build-frontend:
	cd frontend && npm run build

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

dev-cli:
	cd cli && go run ./cmd/cloudctl

test:
	cd backend && go test ./... -v -race
	cd cli && go test ./... -v

lint:
	cd backend && golangci-lint run ./...
	cd cli && golangci-lint run ./...
	cd frontend && npm run lint

clean:
	rm -rf backend/bin cli/bin frontend/dist
	find . -name "*.db" -delete

docker-dev:
	docker compose -f docker-compose.dev.yml up --build

docker-dev-down:
	docker compose -f docker-compose.dev.yml down -v
