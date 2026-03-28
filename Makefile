.PHONY: run test test-cover lint check-all build-wasm clean arch generate-wardens generate-assets

# Desktop development
run:
	go run ./cmd/game/

# Tests
test:
	go test -race -count=1 ./...

# Tests with coverage
test-cover:
	go test -race -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Lint
lint:
	golangci-lint run ./...

# Full CI check
check-all: lint test
	@echo "All checks passed."

# WASM build
build-wasm:
	mkdir -p dist/web
	GOOS=js GOARCH=wasm go build -o dist/web/game.wasm ./cmd/game/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/web/
	cp web/index.html dist/web/
	@echo "WASM build done. Serve dist/web/"

# Asset generation: visual description JSON → SVG → PNG
generate-wardens:
	node scripts/generate-assets.mjs wardens --force

generate-assets:
	node scripts/generate-assets.mjs all --force

# Architecture visualization (pkg deps + struct diagram + module index)
arch:
	./scripts/gen-arch.sh

# Clean
clean:
	rm -rf dist/ coverage.out
