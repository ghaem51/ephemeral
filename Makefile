SHELL := /bin/sh

COMPOSE := docker compose
CONTROL_PLANE_DIR := $(CURDIR)/apps/control-plane
DEMO_SERVICE_DIR := $(CURDIR)/demo/demo-service
GO_CONTROL := docker run --rm -v "$(CONTROL_PLANE_DIR):/workspace" -w /workspace golang:1.24
GO_DEMO := docker run --rm -v "$(DEMO_SERVICE_DIR):/workspace" -w /workspace golang:1.24

.PHONY: dev build test lint demo-images clean reset

dev: demo-images
	$(COMPOSE) up --build

build: demo-images
	$(COMPOSE) build control-plane web-console

test:
	$(GO_CONTROL) go test ./...
	$(GO_DEMO) go test ./...
	npm --prefix apps/web-console ci
	npm --prefix apps/web-console run typecheck

lint:
	$(GO_CONTROL) sh -c 'test -z "$$(gofmt -l .)" && go vet ./...'
	$(GO_DEMO) sh -c 'test -z "$$(gofmt -l .)" && go vet ./...'
	npm --prefix apps/web-console ci
	npm --prefix apps/web-console run lint

demo-images:
	$(COMPOSE) --profile images build demo-healthy demo-unhealthy

clean:
	$(COMPOSE) down --remove-orphans
	rm -rf apps/web-console/dist

reset:
	$(COMPOSE) down --volumes --remove-orphans
	rm -rf apps/web-console/dist
