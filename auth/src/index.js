import express from "express";
import dotenv from "dotenv";
import bcrypt from "bcrypt";
import jwt from "jsonwebtoken";
import pool, { query } from "./config/database.js";
import {
  connectRabbitMQ,
  publishEvent,
  closeRabbitMQ,
} from "./config/rabbitmq.js";
import {
  registerValidation,
  loginValidation,
  uuidValidation,
  validateToken,
} from "./middleware/validators.js";
import logger from "./config/logger.js";
import httpLogger from "./middleware/httpLogger.js";

dotenv.config();

const app = express();
app.use(express.json());
app.use(httpLogger);

const PORT = process.env.PORT || 3001;
const JWT_SECRET = process.env.JWT_SECRET || "your-secret-key";
const JWT_EXPIRES_IN = process.env.JWT_EXPIRES_IN || "24h";

// Health check endpoint
app.get("/health", (req, res) => {
  res.json({ status: "healthy", service: "auth" });
});

// Register endpoint
app.post("/api/auth/register", registerValidation, async (req, res) => {
  const { username, email, password } = req.body;

  try {
    // Hash password
    const password_hash = await bcrypt.hash(password, 10);

    // Insert user
    const result = await query(
      "INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, username, email, created_at",
      [username, email, password_hash]
    );

    const user = result.rows[0];

    // Publish user.created event to RabbitMQ
    await publishEvent("user.created", {
      userId: user.id,
      username: user.username,
      email: user.email,
      createdAt: user.created_at,
    });

    logger.info(`User registered: ${user.username} (${user.id})`);

    res.status(201).json({
      message: "User registered successfully",
      user: {
        id: user.id,
        username: user.username,
        email: user.email,
      },
    });
  } catch (error) {
    logger.error(`Registration error: ${error.message}`, {
      stack: error.stack,
    });
    if (error.code === "23505") {
      return res
        .status(409)
        .json({ error: "Username or email already exists" });
    }
    res.status(500).json({ error: "Internal server error" });
  }
});

// Login endpoint
app.post("/api/auth/login", loginValidation, async (req, res) => {
  const { email, password } = req.body;

  try {
    const result = await query("SELECT * FROM users WHERE email = $1", [email]);

    if (result.rows.length === 0) {
      return res.status(401).json({ error: "Invalid credentials" });
    }

    const user = result.rows[0];

    // Verify password
    const isValid = await bcrypt.compare(password, user.password_hash);

    if (!isValid) {
      return res.status(401).json({ error: "Invalid credentials" });
    }

    // Generate JWT token
    const token = jwt.sign(
      { userId: user.id, username: user.username, email: user.email },
      JWT_SECRET,
      { expiresIn: JWT_EXPIRES_IN }
    );

    // Publish user.login event
    await publishEvent("user.login", {
      userId: user.id,
      username: user.username,
      loginAt: new Date(),
    });

    logger.info(`User logged in: ${user.username} (${user.id})`);

    res.json({
      message: "Login successful",
      token,
      user: {
        id: user.id,
        username: user.username,
        email: user.email,
      },
    });
  } catch (error) {
    logger.error(`Login error: ${error.message}`, { stack: error.stack });
    res.status(500).json({ error: "Internal server error" });
  }
});

// Verify token endpoint
app.get("/api/auth/verify", validateToken, async (req, res) => {
  const token = req.headers.authorization.split(" ")[1];

  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    res.json({ valid: true, user: decoded });
  } catch (error) {
    res.status(401).json({ error: "Invalid token" });
  }
});

// Get user by ID
app.get("/api/auth/users/:id", uuidValidation, async (req, res) => {
  try {
    const result = await query(
      "SELECT id, username, email, created_at FROM users WHERE id = $1",
      [req.params.id]
    );

    if (result.rows.length === 0) {
      return res.status(404).json({ error: "User not found" });
    }

    res.json({ user: result.rows[0] });
  } catch (error) {
    logger.error(`Get user error: ${error.message}`, { stack: error.stack });
    res.status(500).json({ error: "Internal server error" });
  }
});

// Graceful shutdown
process.on("SIGTERM", async () => {
  logger.info("SIGTERM received, closing connections...");
  await pool.end();
  await closeRabbitMQ();
  process.exit(0);
});

process.on("SIGINT", async () => {
  logger.info("SIGINT received, closing connections...");
  await pool.end();
  await closeRabbitMQ();
  process.exit(0);
});

// Start server
async function start() {
  try {
    // Connect to RabbitMQ
    await connectRabbitMQ();

    // Test database connection
    await query("SELECT NOW()");
    logger.info("Database connected successfully");

    app.listen(PORT, () => {
      logger.info(`Auth Service is running on port ${PORT}`);
      logger.info(`Environment: ${process.env.NODE_ENV || "development"}`);
    });
  } catch (error) {
    logger.error(`Failed to start server: ${error.message}`, {
      stack: error.stack,
    });
    process.exit(1);
  }
}

start();
