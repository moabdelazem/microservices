import logger from "../config/logger.js";

/**
 * HTTP request logging middleware using Winston only
 * Logs method, URL, status code, response time, and user ID
 */
const httpLogger = (req, res, next) => {
  const start = Date.now();

  // Capture original end function
  const originalEnd = res.end;

  // Override res.end to log after response is sent
  res.end = function (...args) {
    // Calculate response time
    const duration = Date.now() - start;

    // Get user ID if available (set by auth middleware)
    const userId = req.user?.userId || "anonymous";

    // Skip health check logs in production
    if (process.env.NODE_ENV !== "production" || req.url !== "/health") {
      // Determine log level based on status code
      const logLevel =
        res.statusCode >= 500
          ? "error"
          : res.statusCode >= 400
          ? "warn"
          : "http";

      // Log the request
      logger[logLevel](
        `${req.method} ${req.url} ${res.statusCode} ${duration}ms - ${userId}`
      );
    }

    // Call original end function
    originalEnd.apply(res, args);
  };

  next();
};

export default httpLogger;
