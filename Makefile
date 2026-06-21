BINARY=bin/secrethub
VERSION=$(shell git describe --tags --always 2>/dev/null || echo "dev")

.PHONY: build build-arm64 test lint clean run

build:
	CGO_ENABLED=0 go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) ./cmd/secrethub

build-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY)-arm64 ./cmd/secrethub

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) $(BINARY)-arm64 coverage.out coverage.html

run: build
	./$(BINARY) serve
