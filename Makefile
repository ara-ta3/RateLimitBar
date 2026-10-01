BIN := bin/ratelimitbar

.PHONY: build test run

build:
	go build -o $(BIN) ./cmd/app

test:
	go test ./...

run: build
	./$(BIN)
