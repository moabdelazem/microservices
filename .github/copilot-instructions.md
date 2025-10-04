# Microservices Project - AI Agent Instructions

## Architecture Overview

This is a **polyglot microservices architecture** with event-driven communication:

- **Auth Service** (Node.js/Express) - Port 3001 - User authentication with JWT
- **Tasks Service** (Go/Gin) - Port 3002 - Task management with JWT auth
- **Client** (React/TypeScript/Vite) - Port 5173 - Frontend (currently minimal/boilerplate)
- **RabbitMQ** - Message broker for cross-service events (topic exchange: `auth_events`)
- **PostgreSQL** - Separate databases per service (`auth_db` on 5432, `tasks_db` on 5433)

### Key Architectural Patterns

1. **Event-Driven User Sync**: Auth service publishes `user.created` events to RabbitMQ. Tasks service consumes these to maintain a local user cache for JWT validation without cross-service calls.

2. **Shared JWT Secret**: Both services use the same `JWT_SECRET` environment variable to validate tokens. The auth service generates tokens; tasks service validates them independently.

3. **Database-per-Service**: Each microservice owns its database. No direct database sharing. Communication only via events or API calls.

4. **Docker Compose Orchestration**:
   - `compose.yml` - Full production stack (all services + dependencies)
   - `compose.dev.yml` - Development dependencies only (RabbitMQ, no app services)
   - Service-specific `compose.dev.yml` in `auth/` and `tasks/` for local dev databases

## Development Workflows

### Starting Services

**Full stack (all services):**

```bash
docker compose up -d
```

**Development mode (run services locally, dependencies in Docker):**

```bash
# Terminal 1: Start shared RabbitMQ
docker compose -f compose.dev.yml up -d

# Terminal 2: Auth service with hot reload
cd auth && npm run watch

# Terminal 3: Tasks service with hot reload
cd tasks && make dev  # Requires 'air' for hot reload, or use 'make run'

# Terminal 4: Client dev server
cd client && npm run dev
```

### Testing

**Integration tests** (tests full auth → tasks flow):

```bash
./scripts/test-integration.sh
```

Requires both services running. Tests: health checks → register → login → create task → verify task ownership.

**Auth service tests:**

```bash
./scripts/test-auth.sh
```

### Building

**Go service:**

```bash
cd tasks && make build  # Outputs to bin/tasks-service
```

**Client:**

```bash
cd client && npm run build
```

## Service-Specific Conventions

### Auth Service (Node.js)

- **ES Modules**: Uses `"type": "module"` - all imports use `.js` extension even for local files
- **Logging**: Winston with daily rotate logs (`auth/logs/`) - use `logger.info()`, `logger.error()`, not `console.log()`
- **Validation**: Uses `express-validator` middleware - see `middleware/validators.js`
- **Password Hashing**: bcrypt with salt rounds 10
- **JWT Claims**: `{ userId, username, email }` with configurable expiry (default 24h)
- **RabbitMQ Publishing**: All user events use routing key pattern `user.{action}` (e.g., `user.created`)

### Tasks Service (Go)

- **Project Structure**: Standard Go layout - `cmd/api/main.go` (entry), `internal/` (private packages), `pkg/` (public utils)
- **Gin Framework**: Production mode controlled by `ENV=production` (not `GIN_MODE`)
- **Middleware Order**: Recovery → Logger → CORS → Auth (on protected routes only)
- **UUID Handling**: Uses `github.com/google/uuid` for all ID operations
- **RabbitMQ Consumer**: Auto-retry connection with exponential backoff (10 attempts default)
- **User Cache**: Tasks service maintains a `users` table synced from auth events - never directly queries auth database
- **Graceful Shutdown**: Implements signal handling for clean shutdown of HTTP server and RabbitMQ connections

### Client (React/TypeScript)

- **Current State**: Minimal Vite boilerplate - NOT yet integrated with backend
- **Planned Integration**: Will communicate with auth (3001) and tasks (3002) services
- **CORS**: Backend services already configured for `http://localhost:5173` origin

## Cross-Service Communication

### JWT Authentication Flow

1. Client → Auth `/api/auth/login` → receives JWT token
2. Client → Tasks `/api/tasks` with `Authorization: Bearer <token>`
3. Tasks service validates JWT using shared `JWT_SECRET`
4. Tasks service checks user exists in local cache (synced via RabbitMQ)

### RabbitMQ Event Flow

**Exchange**: `auth_events` (type: topic, durable)

**Events Published by Auth:**

- `user.created` - payload: `{ userId, username, email, createdAt }`

**Events Consumed by Tasks:**

- Binds to `user.#` pattern
- Inserts/updates user in local cache on `user.created`

### Database Schemas

**Auth `users` table:**

```sql
id UUID PRIMARY KEY, username VARCHAR(50) UNIQUE, email VARCHAR(100) UNIQUE,
password_hash TEXT, created_at TIMESTAMP
```

**Tasks `users` table** (cache only):

```sql
id UUID PRIMARY KEY, username VARCHAR(50), email VARCHAR(100),
created_at TIMESTAMP, synced_at TIMESTAMP
```

**Tasks `tasks` table:**

```sql
id UUID PRIMARY KEY, user_id UUID REFERENCES users(id), title VARCHAR(200),
description TEXT, status VARCHAR(20), created_at TIMESTAMP, updated_at TIMESTAMP
```

## Environment Variables

**Critical shared variables** (must match across services):

- `JWT_SECRET` - Used by both auth and tasks for token generation/validation
- `RABBITMQ_URL` - Default: `amqp://admin:admin@localhost:5672`

**Service-specific**: See individual service READMEs or `.env.example` files (not yet created - extract from compose files)

## Common Pitfalls

1. **Port Conflicts**: Auth DB (5432), Tasks DB (5433) - don't confuse them
2. **JWT Secret Mismatch**: If tasks can't validate tokens, check `JWT_SECRET` matches auth service
3. **RabbitMQ Not Running**: Tasks service needs RabbitMQ for user sync - fails to start without it
4. **ES Module Imports**: In auth service, always use `.js` extension: `import x from './file.js'`
5. **User Not Found in Tasks**: Ensure user was created in auth AFTER tasks service started listening to events

## File Patterns

- **Go internal packages**: Only accessible within tasks service - don't try to import from other services
- **Docker init scripts**: `init.sql` files in service roots - auto-run on first database container start
- **Compose files**: Root level for full stack, service level for local dev dependencies
