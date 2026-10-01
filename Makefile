BIN := bin/ratelimitbar

build:
	go build -o $(BIN) ./cmd/app

test:
	go test ./...

fmt:
	go fmt ./...

fmt/check: fmt
	git diff --exit-code -- '*.go'

run: build
	./$(BIN)
