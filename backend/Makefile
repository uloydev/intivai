.PHONY: check lint vet build test coverage test-integration test-integration-dev migrate dev smoke

# Full pre-commit gate (unit-level)
check: lint vet build test

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

dev:
	DOCKER_BUILDKIT=0 docker build -t intivai-app:latest .
	cd .. && docker compose --env-file .env -f docker-compose.yml -f docker-compose.dev.yml up -d

CV_PDF ?= /tmp/kilo/cv.pdf

smoke:
	bash ../scripts/smoke.sh $(CV_PDF)
