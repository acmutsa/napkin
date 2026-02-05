# Napkin

A monorepo containing a React frontend and Go backend.

## Structure

```
napkin/
├── napkin-app/      # Vite + React + TypeScript frontend
├── napkin-backend/  # Go backend
├── package.json     # Root package.json
└── pnpm-workspace.yaml
```

## Getting Started

### Prerequisites

- Node.js >= 18
- pnpm >= 8
- Go >= 1.21

### Install Dependencies

```bash
pnpm install
```

## Development

Run both frontend and backend with a single command:
reat
```bash
pnpm dev
```

- Frontend: `http://localhost:5173` (with API proxy to backend)
- Backend: `http://localhost:8080`

In development, the Vite dev server proxies `/api/*` and `/health` requests to the Go backend, so you only need to access `http://localhost:5173`.

### Individual Commands

```bash
pnpm dev:frontend  # Frontend only
pnpm dev:backend   # Backend only
```

## Production

Build both frontend and backend:

```bash
pnpm build
```

This builds the React app to `napkin-app/dist/` and compiles the Go binary to `napkin-backend/bin/server`.

Start the production server:

```bash
pnpm start
```

The Go server serves both the API and static frontend files on `http://localhost:8080`.

## API Endpoints

| Endpoint     | Method | Description         |
| ------------ | ------ | ------------------- |
| `/health`    | GET    | Health check        |
| `/api/hello` | GET    | Hello world example |
