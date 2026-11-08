export GOCACHE := $(CURDIR)/.cache/go-build
export GOMODCACHE := $(CURDIR)/.cache/go-mod
COMPOSE ?= docker compose

.PHONY: up down logs test integration eval eval-live check
up:
	$(COMPOSE) up -d --build --wait api

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs --tail=50 api init

test:
	go test ./...

integration:
	@test -n "$$TEST_DATABASE_URL" || (echo 'Set TEST_DATABASE_URL to a local PostgreSQL database'; exit 1)
	go test -count=1 -v ./tests/integration

eval:
	python3 scripts/evaluate.py

eval-live:
	python3 scripts/evaluate.py --live

check:
	go vet ./...
	go test ./...

.PHONY: companion package knowledge-check
companion:
	npm ci --prefix apps/companion/web --no-audit --no-fund
	npm run build --prefix apps/companion/web
	go run ./apps/companion

package:
	python3 scripts/package_companion.py

knowledge-check:
	python3 scripts/validate_knowledge.py

.PHONY: e2e
e2e:
	go build -o .cache/jornia-companion ./apps/companion
	WISP_TEST_EXPORT_PATH="$(CURDIR)/.cache/addon-export.txt" go test -count=1 -run TestAddonExport ./internal/companion
	npm ci --prefix tests/e2e --no-audit --no-fund
	npm test --prefix tests/e2e

.PHONY: download-page
download-page:
	python3 scripts/build_download.py
