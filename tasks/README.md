# Tasks Service (Go)

A high-performance tasks microservice built with Go, Gin, PostgreSQL, and RabbitMQ.

## Features

- ✅ RESTful API for task management (CRUD operations)
- ✅ JWT authentication with shared secret
- ✅ User data synchronization via RabbitMQ
- ✅ PostgreSQL for data persistence
- ✅ Task filtering and pagination
- ✅ Task statistics endpoint
- ✅ Graceful shutdown handling
- ✅ Production-ready with proper error handling

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL 16
- **Message Broker**: RabbitMQ 3.13
- **Authentication**: JWT (golang-jwt/jwt)

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 16
- RabbitMQ 3.13
- Auth service running (for JWT token generation)

## Installation

1. Install dependencies:

```bash
go mod download
```

2. Copy environment file:

```bash
cp .env.example .env
```

3. Start PostgreSQL (via Docker Compose):

```bash
docker compose -f compose.dev.yml up -d
```

4. Run the service:

```bash
go run cmd/api/main.go
```

Or build and run:

```bash
go build -o tasks cmd/api/main.go
./tasks
```

## API Endpoints

### Public

- `GET /health` - Health check

### Protected (Requires JWT)

- `POST /api/tasks` - Create a new task
- `GET /api/tasks` - List all tasks (with filters)
- `GET /api/tasks/:id` - Get a specific task
- `PUT /api/tasks/:id` - Update a task
- `DELETE /api/tasks/:id` - Delete a task
- `GET /api/tasks/stats/summary` - Get task statistics

## Environment Variables

See `.env.example` for all configuration options.

## Project Structure

```
tasks/
├── cmd/
│   └── api/
│       └── main.go          # Application entry point
├── internal/
│   ├── database/
│   │   └── database.go      # PostgreSQL connection
│   ├── handlers/
│   │   └── tasks.go         # HTTP handlers
│   ├── middleware/
│   │   ├── auth.go          # JWT authentication
│   │   └── logger.go        # HTTP logging
│   ├── models/
│   │   └── models.go        # Data models
│   └── rabbitmq/
│       └── consumer.go      # RabbitMQ consumer
├── .env                     # Environment variables
├── compose.dev.yml          # Docker Compose for PostgreSQL
├── init.sql                 # Database schema
└── go.mod                   # Go dependencies
```

## Testing

```bash
# From the root microservices directory
./scripts/test-tasks.sh
```

## Docker

Build image:

```bash
docker build -t tasks-service:latest .
```

## License

MIT
