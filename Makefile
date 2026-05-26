.PHONY: build test install

build:
	go build -o bin/ibkr2tv ./cmd/ibkr2tv

test:
	go test ./...

install:
	go install ./cmd/ibkr2tv
