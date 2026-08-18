.PHONY: check lint vet build test proto sandbox-images coverage test-integration test-integration-dev migrate dev redis-clear smoke load-ws seed backup restore load-k6

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

