.PHONY: all lint lintall vet

all: vet lint

lint:
	golangci-lint run

lintall:
	golangci-lint run --default all --disable wsl || true

test:
	go test

vet:
	go vet
