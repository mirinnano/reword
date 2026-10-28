SHELL := /bin/bash
-include .env
export

.PHONY: up down migrate api web gen check test lint

up:            ## start local MariaDB
	docker compose up -d --wait mariadb

down:
	docker compose down

migrate:       ## apply migrations
	cd api && go run ./cmd/migrate up

api:           ## run the Go API on :8080
	cd api && go run ./cmd/api

web:           ## run Next.js on :3000 (proxies /api to the Go API)
	cd web && pnpm dev

gen:           ## regenerate sqlc code
	cd api && go tool sqlc generate

test:          ## Go tests (DB tests run when DATABASE_DSN is set)
	cd api && go test ./...

lint:
	cd api && go vet ./... && go tool sqlc diff
	cd web && pnpm lint

check: lint test
	cd web && pnpm build
