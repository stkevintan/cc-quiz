.PHONY: backend frontend dev build-app validate lint-backend lint-frontend build-backend build-frontend test-backend install-deps

LOCAL_GO := $(CURDIR)/.tools/go/bin/go
GO := $(if $(wildcard $(LOCAL_GO)),$(LOCAL_GO),go)
LOCAL_GOFMT := $(CURDIR)/.tools/go/bin/gofmt
GOFMT := $(if $(wildcard $(LOCAL_GOFMT)),$(LOCAL_GOFMT),gofmt)

backend:
	$(GO) run ./cmd/server

frontend:
	npm run dev

dev:
	./scripts/dev.sh

install-deps:
	$(GO) mod download
	npm ci

build-app:
	./scripts/build.sh

validate: lint-backend lint-frontend test-backend build-frontend build-backend build-app

build-backend:
	$(GO) build ./cmd/server

build-frontend:
	npm run build

test-backend:
	$(GO) test ./...

lint-backend:
	$(GO) vet ./... && out=`find . -name '*.go' -print0 | xargs -0 $(GOFMT) -l`; if [ -n "$$out" ]; then printf '%s\n' "$$out"; exit 1; fi

lint-frontend:
	npm run lint
