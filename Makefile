.DEFAULT_GOAL := build

fmt:
	@echo "==> Formatting code"
	@go fmt ./...

.PHONY: fmt

lint: fmt
	@echo "==> Linting code"
	@golint ./...

.PHONY: lint

vet: lint
	@echo "==> Vetting code"
	@golangci-lint run

.PHONY: vet

build: vet
	@echo "==> Building binary"
	@go build main.go

.PHONY: build