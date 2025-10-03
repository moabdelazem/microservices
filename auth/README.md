# Auth Service

Authentication microservice with PostgreSQL and RabbitMQ integration.

## Features

- User registration with password hashing (bcrypt)
- User login with JWT token generation
- Token verification endpoint
- PostgreSQL database with automatic schema initialization
- RabbitMQ event publishing for user actions
- Health check endpoint

## Prerequisites

- Node.js 18+
- Docker & Docker Compose

## Setup

1. Install dependencies:

```bash
npm install
```

2. Start PostgreSQL and RabbitMQ:

```bash
npm run compose-dev
```

The `init.sql` file will automatically create the users table when the PostgreSQL container starts.

3. Start the service:

```bash
npm run watch  # Development with auto-reload
# or
npm start      # Production
```

## API Endpoints

### Health Check

```
GET /health
```

### Register User

```
POST /api/auth/register
Content-Type: application/json

{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

### Login

```
POST /api/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword"
}
```

### Verify Token

```
GET /api/auth/verify
Authorization: Bearer <token>
```

### Get User by ID

```
GET /api/auth/users/:id
```

## Database Schema

The service automatically creates the following schema on startup:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## RabbitMQ Events

The service publishes the following events:

- `user.created` - When a new user registers
- `user.login` - When a user logs in

## Environment Variables

See `.env.example` for all configuration options.

## Services

- **Auth Service**: http://localhost:3001
- **PostgreSQL**: localhost:5432
- **RabbitMQ Management UI**: http://localhost:15673 (admin/admin)
- **RabbitMQ AMQP**: localhost:5673
