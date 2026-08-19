.DEFAULT_GOAL := help
.PHONY: help check lint vet build test proto sandbox-images coverage test-integration test-integration-dev migrate dev redis-clear smoke load-ws seed backup restore load-k6

# Display available commands and descriptions
help:
	@printf "\n\033[1;36mIntivai Backend — Commands & Targets\033[0m\n\n"
	@printf "\033[1mUsage:\033[0m make <target> [OPTIONS]\n\n"
	@printf "\033[1;33mBuild & Quality Gates:\033[0m\n"
	@printf "  \033[32mcheck\033[0m                 Run lint, vet, build, and unit tests\n"
	@printf "  \033[32mlint\033[0m                  Run golangci-lint (.golangci.yml)\n"
	@printf "  \033[32mvet\033[0m                   Run go vet ./...\n"
	@printf "  \033[32mbuild\033[0m                 Compile backend binary\n"
	@printf "  \033[32mtest\033[0m                  Run Go unit tests\n"
	@printf "  \033[32mcoverage\033[0m              Verify per-package coverage floors (domain >=70%%, others >=50%%)\n\n"
	@printf "\033[1;33mSandbox & Protobuf:\033[0m\n"
	@printf "  \033[32mproto\033[0m                 Regenerate sandbox gRPC stubs from proto/sandbox.proto\n"
	@printf "  \033[32msandbox-images\033[0m        Build the 4 language sandbox images (Go, Python, Node, TS)\n\n"
	@printf "\033[1;33mDatabase & Seed Scenarios:\033[0m\n"
	@printf "  \033[32mmigrate\033[0m               Apply pending PostgreSQL migrations\n"
	@printf "  \033[32mseed [SCENARIO=demo]\033[0m  Apply modular SQL seed scenario (e.g. 'make seed demo')\n"
	@printf "  \033[32mseed-fresh [SCENARIO]\033[0m Reset volumes, recreate containers, migrate & seed\n"
	@printf "  \033[32mbackup\033[0m                Create PostgreSQL and MinIO backup archive\n"
	@printf "  \033[32mrestore DUMP=...\033[0m      Restore database from backup dump\n\n"
	@printf "\033[1;33mTesting & Benchmarks:\033[0m\n"
	@printf "  \033[32mtest-integration-dev\033[0m  Run integration tests against local dev compose stack\n"
	@printf "  \033[32mtest-integration\033[0m      Run integration tests against custom TEST_DATABASE_URL\n"
	@printf "  \033[32msmoke [CV_PDF=...]\033[0m    Run end-to-end API scenario against running stack\n"
	@printf "  \033[32mload-ws\033[0m               Run 100-concurrent WebSocket load check (CONNS overrides)\n"
	@printf "  \033[32mload-k6\033[0m               Run k6 REST load test (100 concurrent virtual users)\n\n"
	@printf "\033[1;33mDevelopment Stack:\033[0m\n"
	@printf "  \033[32mdev\033[0m                   Boot full stack on fresh Redis (builds app, sandbox & certs)\n"
	@printf "  \033[32mredis-clear\033[0m           Flush Redis queue while stack stays running\n\n"

# Full pre-commit gate (unit-level)
check: lint vet build test

# Regenerate the sandbox gRPC stubs from proto/sandbox.proto
proto:
	protoc --go_out=paths=source_relative:internal/sandbox/proto \
		--go-grpc_out=paths=source_relative:internal/sandbox/proto \
		--proto_path=proto proto/sandbox.proto

# Build the sandbox execution images + the sidecar (ADR-0002). Run once per
# machine (or after Dockerfile changes); the sidecar pulls them by tag.
sandbox-images:
	DOCKER_BUILDKIT=0 docker build -t intivai-sandbox-go ./docker/sandbox/go
	DOCKER_BUILDKIT=0 docker build -t intivai-sandbox-python ./docker/sandbox/python
	DOCKER_BUILDKIT=0 docker build -t intivai-sandbox-node ./docker/sandbox/node
	DOCKER_BUILDKIT=0 docker build -t intivai-sandbox-ts ./docker/sandbox/ts

lint:
	golangci-lint run ./...

vet:
	go vet ./...

build:
	go build ./...

test:
	go test ./...

# Coverage gate: per-package floors (see scripts/check-coverage.sh).
# Needs the dev stack up (integration-gated tests must count).
coverage:
	TEST_DATABASE_URL='postgres://intivai_app:intivai_app@localhost:5433/intivai?sslmode=disable' \
	TEST_REDIS_ADDR='localhost:6380' \
	TEST_MINIO_ENDPOINT='localhost:9002' TEST_MINIO_ACCESS='minioadmin' TEST_MINIO_SECRET='minioadmin' \
	bash ../scripts/check-coverage.sh

# Integration tests against a live Postgres+Redis (see CI + docker-compose).
# Pass TEST_DATABASE_URL / TEST_REDIS_ADDR to point elsewhere.
test-integration:
	@test -n "$(TEST_DATABASE_URL)" || (echo "TEST_DATABASE_URL required (e.g. postgres://intivai_app:intivai_app@localhost:5433/intivai?sslmode=disable)" && exit 1)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" TEST_REDIS_ADDR="$(TEST_REDIS_ADDR)" go test ./...

# Integration tests against the local dev stack (docker compose up first)
test-integration-dev:
	TEST_DATABASE_URL='postgres://intivai_app:intivai_app@localhost:5433/intivai?sslmode=disable' \
	TEST_REDIS_ADDR='localhost:6380' \
	TEST_MINIO_ENDPOINT='localhost:9002' TEST_MINIO_ACCESS='minioadmin' TEST_MINIO_SECRET='minioadmin' \
	go test ./...

migrate:
	INTIVAI_MIGRATE_URL='postgres://intivai:intivai@localhost:5433/intivai?sslmode=disable' go run ./cmd/server -migrate-only

# Boot the dev stack on a FRESH redis: stale asynq tasks/dead-letters from a
# previous session must not leak into the new one. Containers are recreated
# (redis data lives in the container layer); postgres + minio volumes persist.
dev:
	cd .. && docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml down
	DOCKER_BUILDKIT=0 docker build -t intivai-app:latest .
	$(MAKE) sandbox-images
	bash ../scripts/gen-sandbox-certs.sh
	cd .. && docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml up -d

# Flush redis while the stack stays up (mid-session queue cleanup).
redis-clear:
	docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml exec redis redis-cli FLUSHALL

CV_PDF ?= /tmp/kilo/cv.pdf

smoke:
	bash ../scripts/smoke.sh $(CV_PDF)

# Support positional scenario argument (e.g. `make seed demo` or `make seed-fresh demo`)
ifeq ($(firstword $(MAKECMDGOALS)),seed)
  SCENARIO_ARG := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(SCENARIO_ARG):;@:)
endif
ifeq ($(firstword $(MAKECMDGOALS)),seed-fresh)
  SCENARIO_ARG := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(SCENARIO_ARG):;@:)
endif

SCENARIO ?= $(if $(SCENARIO_ARG),$(firstword $(SCENARIO_ARG)),demo)

# Seed sample data for local testing (Usage: make seed [demo] or make seed SCENARIO=demo)
seed:
	bash ../scripts/seed.sh $(SCENARIO)

# Wipe volumes, start fresh containers, apply migrations, and seed sample data
seed-fresh:
	cd .. && docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml down -v
	DOCKER_BUILDKIT=0 docker build -t intivai-app:latest .
	$(MAKE) sandbox-images
	bash ../scripts/gen-sandbox-certs.sh
	cd .. && docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml up -d
	sleep 3
	$(MAKE) migrate
	$(MAKE) seed SCENARIO=$(SCENARIO)

# Nightly DB & MinIO backup
backup:
	bash ../scripts/backup.sh

# Disaster recovery restore (Usage: make restore DUMP=intivai-XXXX.sql.gz)
restore:
	bash ../scripts/restore.sh $(DUMP)

# 100-concurrent-WS load check (M3 criterion). Needs the dev stack up.
# CONNS overrides the connection count.
load-ws:
	TEST_DATABASE_URL='postgres://intivai_app:intivai_app@localhost:5433/intivai?sslmode=disable' \
	BASE_URL='http://localhost:8081' CONNS=$(or $(CONNS),100) go run ./cmd/loadcheck

# k6 REST load test (100 concurrent users)
load-k6:
	k6 run ../scripts/k6_load.js

