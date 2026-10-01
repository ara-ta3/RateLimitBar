BIN := bin/ratelimitbar

.PHONY: build test fmt-check run

build:
	go build -o $(BIN) ./cmd/app

test:
	go test ./...

fmt-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "未整形のファイル:"; echo "$$unformatted"; exit 1; \
	fi

run: build
	./$(BIN)
