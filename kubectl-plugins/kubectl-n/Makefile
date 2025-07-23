ARCH=$(shell uname -s | tr '[A-Z]' '[a-z]')
BINARY_NAME=kubectl-n
LD_FLAGS=-X github.com/jim-barber-he/go/util.version=${VERSION}
VERSION=$(shell git log -1 --date=format:%Y.%m.%d --pretty=format:%ad-%h)

.PHONY: all build clean install lint run vet

all: vet lint build install

build:
	GOARCH=amd64 GOOS=darwin go build -ldflags "${LD_FLAGS}" -o ${BINARY_NAME}-darwin -race main.go
	GOARCH=amd64 GOOS=linux go build -ldflags "${LD_FLAGS}" -o ${BINARY_NAME}-linux -race main.go

clean:
	go clean
	rm -f ${BINARY_NAME}-darwin ${BINARY_NAME}-linux

install:
	go install -ldflags="-s ${LD_FLAGS}"

lint:
	golangci-lint run

lintall:
	golangci-lint run --default all --disable depguard,exhaustruct,wsl || true

run: build
	./${BINARY_NAME}-${ARCH}

test:
	go test

vet:
	go vet
