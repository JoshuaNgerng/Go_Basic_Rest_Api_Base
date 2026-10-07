# goose is run via `go run`, so there is nothing to install.
GOOSE      = go run github.com/pressly/goose/v3/cmd/goose@v3.22.1
MIGRATIONS = db/migrations
OAPI       = go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
# loads .env into the shell before running a command
WITH_ENV   = set -a; . ./.env; set +a;

.PHONY: db tidy sqlc generate run migrate-up migrate-down migrate-status migrate-create

db:                      ## start postgres
	docker compose up -d

tidy:
	go mod tidy

migrate-up:              ## apply all pending migrations
	$(WITH_ENV) $(GOOSE) -dir $(MIGRATIONS) postgres "$$DATABASE_URL" up

migrate-down:            ## roll back the last migration
	$(WITH_ENV) $(GOOSE) -dir $(MIGRATIONS) postgres "$$DATABASE_URL" down

migrate-status:          ## show which migrations are applied
	$(WITH_ENV) $(GOOSE) -dir $(MIGRATIONS) postgres "$$DATABASE_URL" status

migrate-create:          ## usage: make migrate-create name=add_phone_to_users
	$(GOOSE) -dir $(MIGRATIONS) create -s $(name) sql

# Regenerates internal/db from db/migrations + db/queries.
# Run after changing a migration or a query.
sqlc:
	sqlc generate

# Regenerates internal/api/api.gen.go from api/openapi.yaml.
# Run after editing the spec.
generate:
	$(OAPI) -config api/oapi-codegen.yaml api/openapi.yaml
	go mod tidy

run:
	go run ./cmd/api
