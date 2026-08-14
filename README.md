# dorms.ng

A secure, hostel marketplace for Nigerian students built for observability and scale.

## The Problem
Nigerian students struggle to find verified, safe, and fairly‑priced hostel accommodation near their campuses. Listings are scattered across unregulated WhatsApp groups, social media, and offline agents leading to scams, price gouging, and wasted time.

## The Solution
`dorms.ng` connects students directly with verified hostel owners. It provides detailed listings and location‑based filtering. Every request is traced, logged, and monitored because housing decisions deserve the same reliability we expect from fintech or healthcare platforms.

## Tech Stack & Key Trade‑Offs

| Component | Choice | Why | Trade‑Off |
|-----------|--------|-----|-----------|
| **Backend** | Go (Chi) | Excellent concurrency (goroutines) for handling concurrent booking requests. Fast compilation and low memory footprint. | More verbose than Python or Node.js but the performance and reliability gain justifies it for a transaction‑heavy domain. |
| **Frontend** | Next.js (React) + TypeScript + Bun | SEO is critical for a marketplace because server‑side rendering (SSR) lets hostel listings appear in search results. Image optimization (next/image) is built‑in for listing photos. Also, using Bun means one more runtime to manage in CI. | Next.js is heavier than a pure SPA (e.g., Vite + React) as deployments take longer. |
| **DB Driver** | `sqlc` | Generates type‑safe Go code from raw SQL. No runtime reflection, no ORM surprises. Full control over complex JOINs. | Loses the dynamic query flexibility of an ORM (e.g., GORM). But for a well‑defined domain like hostels/listings, static queries are a net win for maintainability. |
| **Database** | PostgreSQL | ACID compliance is non‑negotiable for booking transactions. Supports row‑level security (RLS) for future multi‑tenancy. | Higher operational overhead but we need production‑grade durability. |
| **Observability** | OpenTelemetry | Vendor‑neutral instrumentation. We can export to New Relic, Datadog, or self‑hosted Prometheus without changing code. | Requires additional infrastructure to collect/visualise. However, we containerise everything with Docker Compose, making local dev mirrors production. |
| **Authentication** | JWT with refresh rotation | Stateless, scales horizontally without session DB lookups. | JWTs cannot be invalidated server‑side without a denylist we mitigate with short‑lived access tokens (15min) and a Redis denylist for logout. |
| **Deployment** | Docker + GitHub Actions | Reproducible builds and one‑command local dev (`docker‑compose up`). CI enforces tests/lint before merge. | GitHub Actions has a 6‑hour runtime limit, our test suite is fast (<2min), so this hasn't been an issue. |
## Getting Started

### Prerequisites
- [Bun](https://bun.sh/) (for frontend)
- [Go](https://go.dev) (for backend)
- Docker & Docker Compose (optional, for containerized development)
- [Lefthook](https://github.com/evilmartians/lefthook) (Git hook manager)

### Installation

1. Clone the repository
```bash
git clone https://github.com/fagbenjaenoch/dorms-ng.git
cd dorms-ng
```

2. Install frontend dependencies with Bun
```bash
cd apps/frontend
bun install
```

3. Setup Environment Variable for frontend
```env
cp .env.example .env
```

4. Install backend dependencies
```bash
cd ../backend
go mod download
```

5. Set up environment variables
```bash
cp .env.example .env
```

6. Run the development servers

**With Docker (recommended)**
```bash
docker-compose up
```

**Without Docker**
- Frontend: `bun run dev`
- Backend: `task dev`

## Project Structure

```
dorms-ng/
├── apps/
│   ├── frontend/          # Next.js application (Bun runtime)
│   └── backend/           # Go application
├── .github/workflows/     # CI/CD pipelines
├── docker-compose.yml     # Production setup
├── docker-compose.dev.yml # Development setup
└── README.md
```

## Development

### Frontend (with Bun)
```bash
cd apps/frontend
bun run dev
```

### Backend
```bash
cd apps/backend
go run main.go
```

### Testing
```bash
# Frontend tests
cd apps/frontend
bun test

# Backend tests
cd apps/backend
task test

# Run the full suite (unit + integration) with:
go test ./... -race -cover
```

## Deployment

The application is configured for deployment on Vercel (frontend) with a containerized backend. The CI/CD pipeline (GitHub Actions) handles automated builds and tests. Bun's fast startup time makes it an excellent choice for serverless deployment environments.

## Observability (Production‑Ready)

All HTTP requests and DB queries are instrumented with **OpenTelemetry**. 

- **Health:** `/health` returns `200 OK` when the DB is reachable.
- **Traces & Metrics:** Export to New Relic (vendor-agnostic, configurable via env vars).
- **Analytics and User monitoring:** Handled using posthog

## What I think of implementing

- **Async Job Queue:** Introduce **NATS / RabbitMQ** with a go worker pool to handle email confirmations and SMS notifications asynchronously. Currently these happen in‑request, which adds latency (Currently in progress at [#19](https://github.com/fagbenjaenoch/dorms-ng/pull/19)).
- **OpenAPI / Swagger:** Autogenerate OpenAPI specs from Gin routes and serve a Swagger UI, making API exploration instantly accessible to frontend engineers.
- **Cache:** Cache frequently requested searches e.g trending hostels to reduce DB traffic and improve overall performance and UX.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
