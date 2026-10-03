BIN := bin/ratelimitbar
APP := dist/RateLimitBar.app

.PHONY: build app test fmt fmt/check run

build:
	go build -o $(BIN) ./cmd/app

app:
	@test "$$(go env GOOS)" = darwin || { echo "make app requires a macOS target" >&2; exit 1; }
	mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources"
	go build -o "$(APP)/Contents/MacOS/ratelimitbar" ./cmd/app
	cp packaging/macos/Info.plist "$(APP)/Contents/Info.plist"
	cp packaging/macos/AppIcon.icns "$(APP)/Contents/Resources/AppIcon.icns"
	install -m 755 packaging/macos/launch "$(APP)/Contents/MacOS/launch"
	plutil -lint "$(APP)/Contents/Info.plist"

test:
	go test ./...

fmt:
	go fmt ./...

fmt/check: fmt
	git diff --exit-code -- '*.go'

run: build
	./$(BIN)
