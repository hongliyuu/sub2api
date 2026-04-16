#!/usr/bin/env bash
# dev-setup.sh — one-time local development setup for Sub2API
# Run this once before starting development. Safe to re-run.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"

# ── colours ───────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[setup]${NC} $*"; }
warn()  { echo -e "${YELLOW}[warn] ${NC} $*"; }
error() { echo -e "${RED}[error]${NC} $*" >&2; exit 1; }

# ── 1. Check required tools ───────────────────────────────────────────────────
info "Checking required tools..."
command -v go    >/dev/null 2>&1 || error "Go is not installed. Install from https://golang.org"
command -v node  >/dev/null 2>&1 || error "Node.js is not installed. Install from https://nodejs.org"
command -v psql  >/dev/null 2>&1 || error "psql not found. Install PostgreSQL 15+."
command -v redis-cli >/dev/null 2>&1 \
  || warn "redis-cli not found — cannot verify Redis. Make sure Redis is running on :6379."

# ── 2. Install pnpm if missing ────────────────────────────────────────────────
if ! command -v pnpm >/dev/null 2>&1; then
  info "Installing pnpm..."
  npm install -g pnpm
else
  info "pnpm already installed ($(pnpm --version))"
fi

# ── 3. Install frontend dependencies ─────────────────────────────────────────
info "Installing frontend dependencies..."
cd "$FRONTEND"
pnpm install

# ── 4. Check PostgreSQL is reachable ─────────────────────────────────────────
info "Checking PostgreSQL connection..."
PG_USER="${PGUSER:-postgres}"
PG_HOST="${PGHOST:-localhost}"
PG_PORT="${PGPORT:-5432}"
PG_PASSWORD="${PGPASSWORD:-}"

pg_isready -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" >/dev/null 2>&1 \
  || error "PostgreSQL is not running on $PG_HOST:$PG_PORT. Start it and re-run."

# ── 5. Create database if it doesn't exist ───────────────────────────────────
DB_EXISTS=$(psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -tAc \
  "SELECT 1 FROM pg_database WHERE datname='sub2api';" 2>/dev/null || true)
if [ "$DB_EXISTS" != "1" ]; then
  info "Creating database 'sub2api'..."
  psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -c "CREATE DATABASE sub2api;" >/dev/null
else
  info "Database 'sub2api' already exists"
fi

# ── 6. Create backend/config.yaml if missing ─────────────────────────────────
CONFIG="$BACKEND/config.yaml"
if [ ! -f "$CONFIG" ]; then
  info "Creating backend/config.yaml from example..."
  cp "$ROOT/deploy/config.example.yaml" "$CONFIG"

  # Patch database section for local dev using Python (avoids sed portability issues)
  python3 - "$CONFIG" "$PG_HOST" "$PG_PORT" "$PG_USER" "$PG_PASSWORD" <<'PYEOF'
import sys, re

config_path, pg_host, pg_port, pg_user, pg_password = sys.argv[1:]

with open(config_path) as f:
    lines = f.readlines()

in_db = False
db_done = False
result = []
for line in lines:
    stripped = line.lstrip()
    if line.rstrip() == 'database:':
        in_db = True
    elif in_db and not db_done:
        if re.match(r'^[a-z]', line) and 'database:' not in line:
            in_db = False
            db_done = True
        elif stripped.startswith('host:'):
            line = f'  host: "{pg_host}"\n'
        elif stripped.startswith('port:'):
            line = f'  port: {pg_port}\n'
        elif stripped.startswith('user:'):
            line = f'  user: "{pg_user}"\n'
        elif stripped.startswith('password:'):
            line = f'  password: "{pg_password}"\n'
        elif stripped.startswith('sslmode:'):
            line = '  sslmode: "disable"\n'
    result.append(line)

with open(config_path, 'w') as f:
    f.writelines(result)
PYEOF

  info "Config written to backend/config.yaml"
else
  info "backend/config.yaml already exists — skipping"
fi

# ── 7. Run DB migrations (start backend briefly to apply Ent migrations) ──────
TABLES_EXIST=$(psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -d sub2api \
  -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='users';" 2>/dev/null || echo "")

if [ "$TABLES_EXIST" != "1" ]; then
  info "Running database migrations (starting backend briefly)..."
  cd "$BACKEND"
  go run ./cmd/server &
  BACKEND_PID=$!
  # Wait until the server is up or 15s, then stop it
  for i in $(seq 1 30); do
    sleep 1
    if curl -sf http://localhost:8080/api/v1/settings/public >/dev/null 2>&1; then
      break
    fi
  done
  kill "$BACKEND_PID" 2>/dev/null || true
  wait "$BACKEND_PID" 2>/dev/null || true
  sleep 1
fi

# ── 8. Create admin user if not present ──────────────────────────────────────
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

ADMIN_EXISTS=$(psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -d sub2api \
  -tAc "SELECT 1 FROM users WHERE email='$ADMIN_EMAIL' AND deleted_at IS NULL;" 2>/dev/null || echo "")

if [ "$ADMIN_EXISTS" != "1" ]; then
  info "Creating admin user ($ADMIN_EMAIL)..."

  # Generate bcrypt hash via Go to match the app's algorithm exactly
  GENHASH_DIR=$(mktemp -d)
  cat > "$GENHASH_DIR/main.go" <<'GOEOF'
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(string(hash))
}
GOEOF
  # Reuse the backend's go.mod so golang.org/x/crypto is already available
  HASH=$(cd "$BACKEND" && go run "$GENHASH_DIR/main.go" "$ADMIN_PASSWORD")
  rm -rf "$GENHASH_DIR"

  psql -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" -d sub2api -c \
    "INSERT INTO users (email, password_hash, role, username, status)
     VALUES ('$ADMIN_EMAIL', '$HASH', 'admin', 'admin', 'active')
     ON CONFLICT DO NOTHING;" >/dev/null

  info "Admin created: $ADMIN_EMAIL / $ADMIN_PASSWORD"
else
  info "Admin user already exists ($ADMIN_EMAIL)"
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}Setup complete!${NC}"
echo ""
echo "  Start backend:   cd backend && go run ./cmd/server"
echo "  Start frontend:  cd frontend && pnpm dev"
echo ""
echo "  Then open:  http://localhost:3000"
echo "  Login:      $ADMIN_EMAIL / $ADMIN_PASSWORD"
echo ""
