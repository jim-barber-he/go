ARCH := $(shell uname -s | tr '[A-Z]' '[a-z]')
BINARY_NAME=kubectl-p

.PHONY: all build clean install lint run vet

all: vet lint build install

build:
	GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin -race main.go
	GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux -race main.go

clean:
	go clean
	rm -f ${BINARY_NAME}-darwin ${BINARY_NAME}-linux

install:
	go install -ldflags='-s'

lint:
	golangci-lint run

lintall:
	golangci-lint run --default all --disable wsl || true

run: build
	./${BINARY_NAME}-${ARCH}

test:
	go test

vet:
	go vet
