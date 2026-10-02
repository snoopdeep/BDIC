.PHONY: help setup deps db demo api web check test build fmt vet clean

# Default target: show what is available.
help:
	@echo "BDIC School Management System"
	@echo ""
	@echo "First time:"
	@echo "  make setup     Install dependencies, create the database, prepare .env"
	@echo ""
	@echo "Every day (two terminals):"
	@echo "  make api       Start the Go API on http://localhost:8080"
	@echo "  make web       Start the Next.js site on http://localhost:3000"
	@echo ""
	@echo "Before committing:"
	@echo "  make check     Format, vet, test, and build both sides"
	@echo ""
	@echo "Individually:"
	@echo "  make deps      Fetch Go modules and npm packages"
	@echo "  make db        Create the local 'bdic' database if it does not exist"
	@echo "  make demo      Insert clearly labelled fictional records for local review"
	@echo "  make test      Run the Go tests"
	@echo "  make fmt       Format the Go code"
	@echo "  make vet       Run go vet"
	@echo "  make build     Compile the API binary and build the frontend"

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

setup: deps db
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ""; \
		echo "Created .env from the template."; \
		echo "Now open .env and fill in AUTH_SIGNING_KEY. Generate one with:"; \
		echo "    openssl rand -base64 48"; \
		echo ""; \
	else \
		echo ".env already exists, leaving it alone."; \
	fi
	@echo "Setup finished. Run 'make api' in one terminal and 'make web' in another."

deps:
	cd backend && go mod tidy
	cd frontend && npm install

# Creates the database only if it is missing, so this is safe to re-run.
db:
	@if psql -lqt 2>/dev/null | cut -d '|' -f 1 | grep -qw bdic; then \
		echo "Database 'bdic' already exists."; \
	else \
		echo "Creating database 'bdic'..."; \
		createdb bdic && echo "Created."; \
	fi

# Idempotent fictional data for demonstrating screens locally. It is never a
# production migration and must not be used with a real school database.
demo:
	psql bdic -v ON_ERROR_STOP=1 -f scripts/demo_data.sql

# ---------------------------------------------------------------------------
# Running
# ---------------------------------------------------------------------------

api:
	cd backend && go run ./cmd/api

web:
	cd frontend && npm run dev

# ---------------------------------------------------------------------------
# Checks
# ---------------------------------------------------------------------------

fmt:
	cd backend && gofmt -l -w ./cmd ./internal

vet:
	cd backend && go vet ./...

test:
	cd backend && go test ./... -count=1

build:
	cd backend && go build -o /dev/null ./cmd/api
	cd frontend && npm run build

# The gate. Run this before every commit.
check: fmt vet test build
	@echo ""
	@echo "All checks passed."

clean:
	cd frontend && rm -rf .next
	rm -rf backend/var/uploads/*
