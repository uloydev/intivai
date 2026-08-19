# Intivai root Makefile — THIN DELEGATION ONLY. No logic, no duplicated
# commands: every backend target forwards to backend/Makefile, frontend
# targets wrap npm scripts. If a command needs real logic, put it in the
# owning Makefile/script, not here.
#
# Docker compose commands live HERE because the compose files live at the
# repo root — backend/Makefile only reaches them via `cd ..`.

.DEFAULT_GOAL := help
.PHONY: help check dev seed seed-fresh migrate smoke redis-clear test-integration-dev coverage backup restore load-ws load-k6 fe-dev fe-run fe fe-build fe-test fe-e2e up down logs ps restart compose-build up-prod down-prod logs-prod ps-prod

# Display available commands and descriptions
help:
	@printf "\n\033[1;36mIntivai — Engineering Workflow & Automation\033[0m\n\n"
	@printf "\033[1mUsage:\033[0m make <target> [OPTIONS]\n\n"
	@printf "\033[1;33mFrontend Development:\033[0m\n"
	@printf "  \033[32mfe-dev\033[0m (or \033[32mfe\033[0m, \033[32mfe-run\033[0m) Start frontend Vite dev server (http://localhost:5173)\n"
	@printf "  \033[32mfe-build\033[0m              Build frontend production bundle (tsc -b + vite build)\n"
	@printf "  \033[32mfe-test\033[0m               Run frontend unit tests (vitest)\n"
	@printf "  \033[32mfe-e2e\033[0m                Run Playwright end-to-end happy path tests\n\n"
	@printf "\033[1;33mQuality & Pre-Commit Gates:\033[0m\n"
	@printf "  \033[32mcheck\033[0m                 Full gate: backend gofmt+lint+vet+build+tests, FE build+vitest\n\n"
	@printf "\033[1;33mDevelopment Stack (Docker Compose):\033[0m\n"
	@printf "  \033[32mdev\033[0m                   Boot full stack on fresh Redis (builds app, sandbox & certs)\n"
	@printf "  \033[32mup\033[0m                    Start dev stack containers (detached)\n"
	@printf "  \033[32mdown\033[0m                  Stop dev stack containers\n"
	@printf "  \033[32mrestart\033[0m               Restart dev stack containers\n"
	@printf "  \033[32mlogs\033[0m                  Follow dev stack container logs\n"
	@printf "  \033[32mps\033[0m                    List running dev stack containers\n"
	@printf "  \033[32mcompose-build\033[0m         Build app image, sandbox images and generate mTLS certs\n"
	@printf "  \033[32mredis-clear\033[0m           Flush Redis queue while stack stays running\n\n"
	@printf "\033[1;33mDatabase & Seed Scenarios:\033[0m\n"
	@printf "  \033[32mmigrate\033[0m               Apply pending PostgreSQL migrations\n"
	@printf "  \033[32mseed [SCENARIO=demo]\033[0m  Apply modular SQL seed scenario (e.g. 'make seed demo')\n"
	@printf "  \033[32mseed-fresh [SCENARIO]\033[0m Reset volumes, recreate containers, migrate & seed\n"
	@printf "  \033[32mbackup\033[0m                Create PostgreSQL and MinIO backup archive\n"
	@printf "  \033[32mrestore DUMP=...\033[0m      Restore database from backup dump\n\n"
	@printf "\033[1;33mTesting & Benchmarks:\033[0m\n"
	@printf "  \033[32mcoverage\033[0m              Check per-package coverage floors (domain >=70%%, others >=50%%)\n"
	@printf "  \033[32mtest-integration-dev\033[0m  Run integration tests against local dev compose stack\n"
	@printf "  \033[32msmoke [CV_PDF=...]\033[0m    Run end-to-end API scenario against running stack\n"
	@printf "  \033[32mload-ws\033[0m               Run 100-concurrent WebSocket load check (CONNS overrides)\n"
	@printf "  \033[32mload-k6\033[0m               Run k6 REST load test (100 concurrent virtual users)\n\n"
	@printf "\033[1;33mProduction Operations (VPS):\033[0m\n"
	@printf "  \033[32mup-prod\033[0m               Deploy and start production stack (--env-file .env.prod)\n"
	@printf "  \033[32mdown-prod\033[0m             Stop production stack\n"
	@printf "  \033[32mlogs-prod\033[0m             Follow production stack logs\n"
	@printf "  \033[32mps-prod\033[0m               List production stack container status\n\n"

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
fe-dev fe-run fe:
	cd frontend && npm run dev

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
