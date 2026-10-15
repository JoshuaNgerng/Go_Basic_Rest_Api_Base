# authapi

A small Go backend with user registration, login, logout, password change and a JWT-protected user profile.

- **Go** `net/http` (Go 1.22+ routing)
- **PostgreSQL** with **sqlc** for type-safe queries
- **goose** for database migrations
- **oapi-codegen** to generate the API types and routes from OpenAPI specs (one spec per domain)
- **PBKDF2-HMAC-SHA256** password hashing and **JWT (HS256)** authentication
- **Swagger UI** at `/docs` for trying the endpoints

## Prerequisites

- Go 1.22 or newer
- Docker (for local Postgres)
- `make`
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)

goose and oapi-codegen run through `go run`, so there is nothing else to install. Their first run downloads them, so it needs internet access.

## Quick start

```bash
cp .env.example .env      # then set JWT_SECRET (e.g. openssl rand -base64 48)
make db                   # start Postgres in Docker
make migrate-up           # create the tables
make sqlc                 # generate internal/db from db/
make generate             # generate api/*/*api/ from api/*/openapi.yaml
make run                  # start the server
```

Open <http://localhost:8080/docs>.

> Generated code is git-ignored (`internal/db` and `api/*/*api/`). After cloning, or after
> changing a spec, a query or a migration, run `make sqlc` and/or `make generate`.

## Configuration

Settings are read from `.env` (if present) and from real environment variables, which take priority.

| Variable            | Default         | Description                            |
| ------------------- | --------------- | -------------------------------------- |
| `PORT`              | `8080`          | HTTP port                              |
| `DATABASE_URL`      | none (required) | Postgres connection string             |
| `JWT_SECRET`        | none (required) | Signing key, at least 32 characters    |
| `JWT_TTL_MINUTES`   | `60`            | Token lifetime                         |
| `PBKDF2_ITERATIONS` | `600000`        | Password hashing cost (minimum 100000) |

## API

| Method | Path                    | Auth | Description                                                |
| ------ | ----------------------- | ---- | ---------------------------------------------------------- |
| POST   | `/auth/register`        | no   | Create an account (`email`, `name`, `password`)            |
| POST   | `/auth/login`           | no   | Returns an `access_token`                                  |
| POST   | `/auth/logout`          | JWT  | Revokes the current token                                  |
| POST   | `/auth/change-password` | JWT  | Needs `current_password` + `new_password`; signs out all sessions |
| GET    | `/users/me`             | JWT  | Returns `email`, `name`, `created_at`, `updated_at` (no primary key) |
| PATCH  | `/users/me`             | JWT  | Update `name` and/or `email` (only sent fields change)     |
| DELETE | `/users/me`             | JWT  | Delete the account                                         |

Try it with curl:

```bash
curl -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","name":"Alice","password":"correct-horse-battery"}'

TOKEN=$(curl -s -X POST localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"correct-horse-battery"}' | jq -r .access_token)

curl localhost:8080/users/me -H "Authorization: Bearer $TOKEN"
curl -X PATCH localhost:8080/users/me -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Alice B"}'

curl -X POST localhost:8080/auth/change-password -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"current_password":"correct-horse-battery","new_password":"even-better-passphrase"}'
curl localhost:8080/users/me -H "Authorization: Bearer $TOKEN"   # now 401: log in again
```

In Swagger UI, use the dropdown at the top to switch between the **Auth** and **User** specs. Log in, copy the token, click **Authorize**, and paste it.

## Project layout

```
api/                              one folder per domain
  embed.go                        embeds the specs so they can be served at /openapi/
  common/                         schemas shared by every domain (Error, Message)
    openapi.yaml
    oapi-codegen.yaml
    commonapi/                    GENERATED
  auth/                           /auth/* operations
    openapi.yaml
    oapi-codegen.yaml
    authapi/                      GENERATED: types, interface, routes
  user/                           /users/* operations
    openapi.yaml
    oapi-codegen.yaml
    userapi/                      GENERATED
cmd/api/main.go                   composition root: builds and wires everything
db/
  migrations/                     goose migrations (also the schema sqlc reads)
  queries/                        SQL queries, one file per topic
internal/
  auth/                           handler.go, service.go, jwt.go, password.go, principal.go
  user/                           handler.go, service.go
  validate/                       input rules shared by several features
  server/                         server.go, router.go, middleware.go, docs.go
  db/                             GENERATED by sqlc
  config/                         .env and environment loading
```

### How a request flows

```
request -> logging/recovery/body-limit -> mux -> route registered by the domain's generated code
        -> requireAuth (only for operations marked `security` in that domain's spec)
        -> generated wrapper decodes the typed request
        -> feature Handler method -> Service -> sqlc Queries -> Postgres
        -> generated code writes the typed response
```

Layer rules:

- **Handlers** (`handler.go`) import their domain's generated package, translate between API types and plain Go values, and map domain errors to responses.
- **Services** (`service.go`) hold the business logic. They do not import generated API packages or `net/http`, and return domain errors such as `ErrEmailTaken`.
- **`internal/db`** is the data layer, generated by sqlc.
- Only `cmd/api/main.go` and `internal/server` know about every package.
- Dependencies point one way: `commonapi` <- `authapi`/`userapi` <- the matching feature package.

### Shared schemas

`Error` and `Message` live in `api/common/openapi.yaml`. Domain specs reference them with
`$ref: "../common/openapi.yaml#/components/schemas/Error"`, and each domain's
`oapi-codegen.yaml` has an `import-mapping` entry that points that path at the generated
`commonapi` package. The key in `import-mapping` must match the `$ref` path exactly.
`securitySchemes` are repeated in each domain spec.

## Common tasks

| Task                         | Command                                       |
| ---------------------------- | --------------------------------------------- |
| Start / stop Postgres        | `make db` / `docker compose down`             |
| Apply migrations             | `make migrate-up`                             |
| Roll back the last migration | `make migrate-down`                           |
| Show migration status        | `make migrate-status`                         |
| New migration                | `make migrate-create name=add_phone_to_users` |
| Regenerate DB code           | `make sqlc`                                   |
| Regenerate API code          | `make generate`                               |
| Run the server               | `make run`                                    |
| Wipe the local database      | `docker compose down -v`                      |

### Add a database change

1. `make migrate-create name=...` and edit the new file.
2. `make migrate-up`.
3. Add or change queries in `db/queries/*.sql`.
4. `make sqlc`.

Never edit a migration that has already been applied anywhere except your own machine; add a new one instead.

### Add an endpoint to an existing domain

1. Add the path, method and a unique `operationId` to that domain's `openapi.yaml` (add `security: - bearerAuth: []` if it needs a token).
2. `make generate`. The build now fails because nothing implements the new operation.
3. Add a method with the matching name and signature to that feature's `Handler`, calling a function on its `Service`.

### Add a new domain (for example `sales`)

1. Create `api/sales/openapi.yaml` and `api/sales/oapi-codegen.yaml` (copy the user ones, change `package`, `output` and the paths).
2. Append `sales` to `API_DOMAINS` in the Makefile and run `make generate`.
3. Create `internal/sales/` with `handler.go` and `service.go`.
4. In `internal/server`: add a block for the new domain in `router.go`, call `NewHandler` in `server.go`, add its `BearerAuthScopes` check to `isProtected` in `middleware.go`, and add its spec to the `urls` list in `docs.go` and to the `//go:embed` line in `api/embed.go`.

Do not edit generated files by hand. Change the spec or the SQL and regenerate.

## Where things belong

- **Credentials** (password, tokens, sessions) belong to the `auth` domain: login, logout, change password.
- **Profile data** (name, email) belongs to the `user` domain.
- A future "forgot password" (email reset token) flow is auth too.

## Security notes

- Passwords are hashed with PBKDF2-HMAC-SHA256, a random 16-byte salt per user, and a stored iteration count, so the cost can be raised later.
- JWTs are signed with HS256, expire after `JWT_TTL_MINUTES`, and only HS256 is accepted.
- JWTs are stateless, so logout works by storing the token ID (`jti`) in `revoked_tokens` until it expires. The middleware checks this table on every protected request, and an hourly job deletes old rows.
- Changing the password sets `users.password_changed_at`. The middleware rejects any token issued before that moment, so all existing sessions end. Deleting a user also invalidates their tokens, because the lookup finds no row.
- Wrong email and wrong password return the same error, and a dummy hash is checked when the email does not exist, so timing does not reveal which emails are registered.
- Unexpected errors are logged and returned to clients as a generic 500.
- Before production: serve over HTTPS, add rate limiting on `/auth/login` and `/auth/change-password`, and keep `JWT_SECRET` out of version control (`.env` is git-ignored).
