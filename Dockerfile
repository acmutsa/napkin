# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY napkin-backend/go.mod ./
RUN go mod download

# Copy source
COPY napkin-backend/ ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Frontend build stage
FROM node:20-alpine AS frontend

RUN corepack enable && corepack prepare pnpm@9.0.0 --activate

WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY napkin-app/package.json ./napkin-app/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy frontend source and build
COPY napkin-app/ ./napkin-app/
RUN pnpm build:frontend

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy frontend dist from frontend stage
COPY --from=frontend /app/napkin-app/dist ./dist

EXPOSE 8080

CMD ["./server"]
