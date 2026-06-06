# 文言文选择题

Classical Chinese quiz app with a Go + Gin + SQLite backend and React + TypeScript + Ant Design frontend.

## Setup

```bash
cp .env.example .env
cp frontend/.env.example frontend/.env
./.tools/go/bin/go version   # optional: verify local Go toolchain
make install-deps
```

`make backend`, `make build-backend`, and `make test-backend` will automatically use `./.tools/go/bin/go` when that local toolchain exists.
On first backend boot the SQLite database is migrated and seeded with an admin, sample teacher/student, one class, and at least 10 active questions.

## Run

```bash
make backend   # API on :8080
make frontend  # Vite on :5173
```

For local full-stack development in one terminal:

```bash
make dev
```

Open http://localhost:5173 and log in with the seeded admin account:

- Username: `admin`
- Password: `admin123`

Seeded demo accounts are also available for local smoke checks: `teacher1` / `teacher123` and `student1` / `student123`.

## Build

To build a single backend binary with the frontend static assets embedded via `go:embed`:

```bash
make build-app
```

That command builds the frontend, copies the generated files into `web/dist`, and produces the embedded binary at `build/classical-chinese-quiz`.

## Validate

```bash
make install-deps
make lint-backend
make lint-frontend
make test-backend
make build-backend
make build-frontend
make build-app
```

`make test-backend` now includes milestone-7 regression coverage for fixed 10-question attempts, single in-progress attempt reuse, first-correct-only scoring, latest wrong-answer replacement, and admin attempt-reset cleanup.
`make build-app` verifies the production packaging path that embeds the frontend bundle into the backend binary.

## GitHub Actions

- `.github/workflows/validate.yml`: runs on pull requests targeting `main` and checks backend lint, frontend lint, backend tests, and frontend/backend builds.
- `.github/workflows/release.yml`: runs when a pull request into `main` is merged, then builds and publishes a Docker image to GitHub Container Registry (`ghcr.io`).
