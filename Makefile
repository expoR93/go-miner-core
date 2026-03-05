# simple Go miner project helpers

.PHONY: all build test bench fmt vet clean

all: build

build:
	go build ./...

test:
	go test ./...

bench:
	go test ./... -bench . -run=^$$

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	go clean ./...
