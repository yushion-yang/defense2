.PHONY: run build test test-cover vet lint check-all build-wasm serve-web android android-aar autoplay autoplay-quick clean arch generate-wardens generate-assets index

# Desktop development
run:
	go run ./cmd/game/

# Desktop binary build
build:
	go build -o dist/defense2 ./cmd/game/
	@echo "Built: dist/defense2"

# Tests
test:
	go test -race -count=1 ./...

# Tests with coverage
test-cover:
	go test -race -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Static analysis
vet:
	go vet ./...

# Lint
lint:
	golangci-lint run ./...

# Full CI check
check-all: vet lint test
	@echo "All checks passed."

# WASM build
build-wasm:
	mkdir -p dist/web
	GOOS=js GOARCH=wasm go build -o dist/web/game.wasm ./cmd/game/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/web/
	cp web/index.html dist/web/
	@echo "WASM build done (dist/web/). Run 'make serve-web' to test."

# Serve WASM build locally
serve-web: build-wasm
	@echo "http://localhost:8080"
	cd dist/web && python3 -m http.server 8080

# Android: build .aar from Go code, then assemble APK via Gradle
android-aar:
	ebitenmobile bind -target android -javapkg com.defense2.game -o android/app/libs/mobile.aar ./cmd/mobile/
	@echo "AAR built: android/app/libs/mobile.aar"

android: android-aar
	cd android && ./gradlew assembleDebug
	@echo "APK: android/app/build/outputs/apk/debug/app-debug.apk"

# Asset generation: visual description JSON → SVG → PNG
generate-wardens:
	node scripts/generate-assets.mjs wardens --force

generate-assets:
	node scripts/generate-assets.mjs all --force

# Architecture visualization (pkg deps + struct diagram + module index)
arch:
	./scripts/gen-arch.sh

# Config viewer (read-only web UI)
preview:
	@echo "Config viewer: http://localhost:8080/web/config-viewer/"
	python3 -m http.server 8080

# Autoplay: full regression sweep (68 games, ~3min)
autoplay:
	go run cmd/autoplay/main.go --sweep --json-dir docs/autotest/M1 --png-dir docs/autotest/M2

# Autoplay: quick smoke test (1 game, ~30s)
autoplay-quick:
	go run cmd/autoplay/main.go --scenario attack-style-coverage

# Project index for AI-assisted development (files/callgraph/configmap)
index:
	go run ./tools/indexer/ -out docs/index

# Clean
clean:
	rm -rf dist/ coverage.out
