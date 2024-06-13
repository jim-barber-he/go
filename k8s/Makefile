ARCH := $(shell uname -s | tr '[A-Z]' '[a-z]')
BINARY_NAME=kubectl-n

.PHONY: all lint lintall vet

all: vet lint

lint:
	golangci-lint run

lintall:
	golangci-lint run --enable-all

vet:
	go vet
