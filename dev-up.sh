#!/usr/bin/env bash
# dev-up.sh — start backend and frontend dev servers
# Run dev-setup.sh first if this is a fresh clone.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND="$SCRIPT_DIR/backend"
FRONTEND="$SCRIPT_DIR/frontend"

GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[dev-up]${NC} $*"; }
error() { echo -e "${RED}[dev-up]${NC} $*" >&2; exit 1; }

# ── Preflight ─────────────────────────────────────────────────────────────────
[ -f "$BACKEND/config.yaml" ] \
  || error "backend/config.yaml not found. Run ./dev-setup.sh first."

lsof -ti :8080 >/dev/null 2>&1 \
  && error "Port 8080 already in use. Stop the existing process first."

lsof -ti :3000 >/dev/null 2>&1 \
  && error "Port 3000 already in use. Stop the existing process first."

# ── Start backend ─────────────────────────────────────────────────────────────
info "Starting backend on :8080 ..."
cd "$BACKEND"
go run ./cmd/server &
BACKEND_PID=$!

# ── Wait for backend to be ready ─────────────────────────────────────────────
for i in $(seq 1 30); do
  sleep 1
  if curl -sf http://localhost:8080/api/v1/settings/public >/dev/null 2>&1; then
    info "Backend ready."
    break
  fi
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    error "Backend process exited unexpectedly. Check output above."
  fi
done

# ── Start frontend ────────────────────────────────────────────────────────────
info "Starting frontend on :3000 ..."
cd "$FRONTEND"
pnpm dev &
FRONTEND_PID=$!

info "Both servers running. Press Ctrl+C to stop."
echo ""
echo "  Frontend:  http://localhost:3000"
echo "  Backend:   http://localhost:8080"
echo ""

# ── Trap Ctrl+C to shut both down cleanly ────────────────────────────────────
trap 'echo ""; info "Stopping..."; kill "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null; wait; info "Done."' INT TERM

wait
