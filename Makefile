VP ?= vp

.PHONY: require-viteplus build test lint format web web-deps web-check hooks renderer-contract test-tui-render generate-contract check-generated openapi-validate architecture-acceptance go-vet race-stateful check ci-support demo demo-smoke docker-fake-smoke arm64-userspace-smoke demo-real-tools-smoke smoke clean run daemon tui api-smoke package schemas openapi
require-viteplus:
	@set -- $(VP); \
	if ! command -v "$$1" >/dev/null 2>&1; then \
		echo "error: required web package manager '$$1' was not found" >&2; \
		echo "install Vite Plus so 'vp' is on PATH, or override VP, e.g. make check VP='corepack pnpm'" >&2; \
		exit 127; \
	fi
build: web-deps
	go build -o flipctld ./cmd/flipctld
	go build -o flipctl-tui ./cmd/flipctl-tui
	cd web && $(VP) build
web: web-deps
	cd web && $(VP) build
web-deps: require-viteplus
	cd web && $(VP) install --frozen-lockfile
test: web-deps
	go test ./...
	cd web && $(VP) test -- --run
lint: web-deps
	cd web && $(VP) lint
format: web-deps
	cd web && $(VP) fmt . --write
web-check: web-deps
	cd web && $(VP) check
hooks:
	git config core.hooksPath .githooks
renderer-contract: web-deps
	go test ./internal/viewdoc -run 'ViewDocument|Renderer|Contract' -count=1
	cd web && $(VP) test -- --run ViewDocument
test-tui-render:
	go test ./internal/tuirender -run 'Renderer|Golden|Noninteractive' -count=1
generate-contract:
	go run ./scripts/generate-contract-ts.go schemas/view-document.json web/src/viewDocumentTypes.ts
check-generated:
	@tmp=$$(mktemp); \
	go run ./scripts/generate-contract-ts.go schemas/view-document.json $$tmp; \
	cmp -s $$tmp web/src/viewDocumentTypes.ts || { echo "generated contract drift: run make generate-contract"; diff -u web/src/viewDocumentTypes.ts $$tmp; rm -f $$tmp; exit 1; }; \
	rm -f $$tmp
openapi-validate:
	go test ./internal/api -run 'OpenAPI' -count=1
architecture-acceptance:
	go test ./internal/api -run 'TestArchitecture' -count=1
go-vet:
	go vet ./...
race-stateful:
	go test -race ./internal/api ./internal/runner -count=1
smoke: build
	./flipctld -addr :18080 & echo $$! > .flipctld.pid; sleep 1; curl -fsS http://localhost:18080/readyz; curl -fsS http://localhost:18080/api/v1/meta; curl -fsS http://localhost:18080/api/v1/apps; curl -fsS http://localhost:18080/api/v1/openapi.json; kill `cat .flipctld.pid`; rm .flipctld.pid
api-smoke: smoke
check: check-generated renderer-contract test-tui-render test build smoke
ci-support: openapi-validate architecture-acceptance go-vet race-stateful
demo:
	docker compose up --build
demo-smoke:
	set -e; \
	docker compose up --build -d; \
	trap 'docker compose down --remove-orphans' EXIT; \
	for i in 1 2 3 4 5 6 7 8 9 10 11 12; do curl -fsS http://localhost:8080/readyz && break || sleep 2; done; \
	curl -fsS http://localhost:8080/api/v1/apps >/dev/null; \
	curl -fsS http://localhost:8080/ >/dev/null

docker-fake-smoke:
	python3 scripts/docker-userspace-smoke.py docker-fake

arm64-userspace-smoke:
	python3 scripts/docker-userspace-smoke.py arm64

demo-real-tools-smoke:
	python3 scripts/demo-real-tools-smoke.py
run daemon:
	go run ./cmd/flipctld -addr :8080
tui:
	go run ./cmd/flipctl-tui -addr http://localhost:8080
schemas openapi:
	@echo "schemas/view-document.json and api/openapi.yaml are checked in; daemon also serves JSON docs"
package: build
	@echo "package skeleton lives in packaging/debian"
clean:
	rm -f flipctld flipctl-tui .flipctld.pid
