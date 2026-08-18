# Intivai root Makefile — THIN DELEGATION ONLY. No logic, no duplicated
# commands: every backend target forwards to backend/Makefile, frontend
# targets wrap npm scripts. If a command needs real logic, put it in the
# owning Makefile/script, not here.
#
# Docker compose commands live HERE because the compose files live at the
# repo root — backend/Makefile only reaches them via `cd ..`.

.PHONY: check dev seed seed-fresh migrate smoke redis-clear test-integration-dev coverage backup restore load-ws load-k6 fe-build fe-test fe-e2e up down logs ps restart compose-build up-prod down-prod logs-prod ps-prod

# Compose invocations (dev = base + dev overlay; prod = base + prod overlay).
# NEVER run prod compose without --env-file .env.prod — base compose
# `environment:` would win and boot with dev secrets.
COMPOSE := docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_PROD := docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml

# Full pre-commit gate: backend lint/vet/build/unit tests + FE typecheck/build + FE unit tests
check:
	$(MAKE) -C backend check
	cd frontend && npm run build
	cd frontend && npx vitest run

# --- Docker compose (dev) ---
up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

restart:
	$(COMPOSE) restart

# Pre-up steps for a fresh machine: app image + sandbox execution images + mTLS certs
compose-build:
	DOCKER_BUILDKIT=0 docker build -t intivai-app:latest ./backend
	$(MAKE) -C backend sandbox-images
	bash scripts/gen-sandbox-certs.sh

# --- Docker compose (prod, VPS) ---
up-prod:
	$(COMPOSE_PROD) up -d

down-prod:
	$(COMPOSE_PROD) down

logs-prod:
	$(COMPOSE_PROD) logs -f

ps-prod:
	$(COMPOSE_PROD) ps

# --- Backend (delegated) ---
dev seed seed-fresh migrate smoke redis-clear test-integration-dev coverage backup restore load-ws load-k6:
	$(MAKE) -C backend $@

# --- Frontend ---
fe-build:
	cd frontend && npm run build

fe-test:
	cd frontend && npx vitest run

# E2E happy path (needs the dev stack up + DeepSeek key)
fe-e2e:
	cd frontend && npx playwright test

# Catch-all: forward any other target (lint, vet, build, test, proto,
# sandbox-images, ...) to the backend Makefile.
%:
	$(MAKE) -C backend $@
