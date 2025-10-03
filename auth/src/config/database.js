import pg from "pg";
import logger from "./logger.js";

const { Pool } = pg;

const pool = new Pool({
  host: process.env.DB_HOST || "localhost",
  port: process.env.DB_PORT || 5432,
  database: process.env.DB_NAME || "auth_db",
  user: process.env.DB_USER || "mo",
  password: process.env.DB_PASSWORD || "mypassword",
  max: 20,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

pool.on("connect", () => {
  logger.debug("PostgreSQL client connected");
});

pool.on("error", (err) => {
  logger.error(`Unexpected PostgreSQL error: ${err.message}`, {
    stack: err.stack,
  });
  process.exit(-1);
});

export const query = (text, params) => pool.query(text, params);

export default pool;
