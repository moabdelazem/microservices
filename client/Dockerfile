# Stage 1: Build the application
FROM node:22-alpine AS builder

# Build arguments for API endpoints (can be overridden at build time)
ARG VITE_AUTH_SERVICE_URL=http://localhost:3001/api/auth
ARG VITE_TASKS_SERVICE_URL=http://localhost:3002/api/tasks

# Set as environment variables for Vite build
ENV VITE_AUTH_SERVICE_URL=$VITE_AUTH_SERVICE_URL
ENV VITE_TASKS_SERVICE_URL=$VITE_TASKS_SERVICE_URL

# Set working directory
WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Stage 2: Serve with Nginx
FROM nginx:alpine

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port 3000 (matching your vite config)
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:3000/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
