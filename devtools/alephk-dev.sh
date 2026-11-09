#!/usr/bin/env bash
# =====================================================================
# AlephK — start everything for development with one command (2026-11-08).
#
#   1) opens the projects in Visual Studio Code
#   2) starts the backends  (usersAppAK :8080, shao :8081), one terminal window each
#   3) waits until the backends answer
#   4) starts the frontends (p3, showcase), one terminal window each
#
# Usage:
#   ./alephk-dev.sh                 everything (VS Code + backends + frontends)
#   ./alephk-dev.sh back            only the backends
#   ./alephk-dev.sh front           only the frontends
#   ./alephk-dev.sh code            only VS Code
#   ./alephk-dev.sh status          what is running (ports)
#   ./alephk-dev.sh stop            stop the backends and the frontends
#   Options: --no-code   --one-window (VS Code: all projects in one window)
#            --dry-run   (show what would run, start nothing)
#
# A program that is already running (its port is busy) is not started twice.
# Closing a terminal window stops that program. No passwords or keys are read
# or printed here: each program reads its own .env as usual (only PORT is
# read from the .env files).
# =====================================================================
set -u

# ---------------------------------------------------------------- settings
GO_ROOT="$HOME/projects/go/src/github.com/milbertk"
REACT_ROOT="$HOME/projects/react/AlephK"

USERS_DIR="$GO_ROOT/usersAppAK"
SHAO_DIR="$GO_ROOT/shao"
P3_DIR="$REACT_ROOT/p3"
SHOWCASE_DIR="$REACT_ROOT/showcase"

# Command that starts each program (go run = always your latest code).
BACKEND_CMD="go run ."
FRONTEND_CMD="npm start"

# Default ports (a PORT=... line in the project's .env wins).
USERS_PORT_DEFAULT=8080
SHAO_PORT_DEFAULT=8081
P3_PORT_DEFAULT=3000
SHOWCASE_PORT_DEFAULT=3001

# Projects opened in VS Code (add security / databasesmng here if you want).
CODE_PROJECTS=("$SHAO_DIR" "$USERS_DIR" "$P3_DIR" "$SHOWCASE_DIR")

# Seconds to wait for the backends before starting the frontends.
BACKEND_WAIT=90
# ---------------------------------------------------------------- end settings

DRY_RUN=0
NO_CODE=0
ONE_WINDOW=0
ACTION="all"
for arg in "$@"; do
  case "$arg" in
    all|back|front|code|status|stop) ACTION="$arg" ;;
    --dry-run) DRY_RUN=1 ;;
    --no-code) NO_CODE=1 ;;
    --one-window) ONE_WINDOW=1 ;;
    -h|--help) sed -n '2,25p' "$0"; exit 0 ;;
    *) echo "Unknown option: $arg (use --help)"; exit 1 ;;
  esac
done

say()  { printf '%s\n' "$*"; }
ok()   { printf '  \033[32m✔\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*"; }
err()  { printf '  \033[31m✖\033[0m %s\n' "$*"; }

# PORT from a project's .env (only that line is read), else the default.
port_of() {
  local dir="$1" default="$2" value=""
  if [ -f "$dir/.env" ]; then
    value=$(grep -E '^[[:space:]]*PORT[[:space:]]*=' "$dir/.env" | tail -n 1 |
      sed -E 's/^[^=]*=[[:space:]]*//; s/[[:space:]]*(#.*)?$//; s/^["'\'']//; s/["'\'']$//')
  fi
  if [[ "$value" =~ ^[0-9]+$ ]]; then echo "$value"; else echo "$default"; fi
}

USERS_PORT=$(port_of "$USERS_DIR" "$USERS_PORT_DEFAULT")
SHAO_PORT=$(port_of "$SHAO_DIR" "$SHAO_PORT_DEFAULT")
P3_PORT=$(port_of "$P3_DIR" "$P3_PORT_DEFAULT")
SHOWCASE_PORT=$(port_of "$SHOWCASE_DIR" "$SHOWCASE_PORT_DEFAULT")

# Is something listening on this local port?
port_busy() {
  (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
}

# PIDs listening on a port (own processes).
port_pids() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltnpH "sport = :$port" 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u
  elif command -v lsof >/dev/null 2>&1; then
    lsof -t -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | sort -u
  fi
}

# Find a terminal program on this computer.
TERMINAL=""
for t in gnome-terminal konsole xfce4-terminal mate-terminal tilix kitty alacritty xterm x-terminal-emulator; do
  if command -v "$t" >/dev/null 2>&1; then TERMINAL="$t"; break; fi
done

# Open a new terminal window that runs a command and stays open afterwards.
#   open_window "<title>" "<folder>" "<command>"
open_window() {
  local title="$1" dir="$2" cmd="$3"
  # Interactive bash (-i) loads ~/.bashrc, so nvm / go are on the PATH.
  local inner="printf '\\033]0;%s\\007' '$title'; cd '$dir' && echo '== $title ==  ($dir)' && echo '\$ $cmd' && trap 'echo' INT && $cmd; trap - INT; echo; echo '== $title stopped. Start it again here with: $cmd   (exit closes the window) =='; exec bash -i"
  if [ "$DRY_RUN" = 1 ]; then
    say "  [dry-run] $TERMINAL window \"$title\": cd $dir && $cmd"
    return 0
  fi
  case "$TERMINAL" in
    gnome-terminal) gnome-terminal --title="$title" -- bash -ic "$inner" >/dev/null 2>&1 & ;;
    konsole)        konsole -p tabtitle="$title" -e bash -ic "$inner" >/dev/null 2>&1 & ;;
    xfce4-terminal) xfce4-terminal --title="$title" -x bash -ic "$inner" >/dev/null 2>&1 & ;;
    mate-terminal)  mate-terminal --title="$title" -- bash -ic "$inner" >/dev/null 2>&1 & ;;
    tilix)          tilix --title="$title" -e "bash -ic \"$inner\"" >/dev/null 2>&1 & ;;
    kitty)          kitty --title "$title" bash -ic "$inner" >/dev/null 2>&1 & ;;
    alacritty)      alacritty --title "$title" -e bash -ic "$inner" >/dev/null 2>&1 & ;;
    xterm)          xterm -T "$title" -e bash -ic "$inner" >/dev/null 2>&1 & ;;
    x-terminal-emulator) x-terminal-emulator -T "$title" -e bash -ic "$inner" >/dev/null 2>&1 & ;;
    *) err "No terminal program found (gnome-terminal, konsole, xterm…)."; return 1 ;;
  esac
  disown 2>/dev/null || true
}

# Start one program in its own window, unless its port is already busy.
#   start_one "<name>" "<folder>" "<port>" "<command>"
start_one() {
  local name="$1" dir="$2" port="$3" cmd="$4"
  if [ ! -d "$dir" ]; then err "$name: folder not found: $dir"; return 1; fi
  if port_busy "$port"; then ok "$name already running on port $port (not started again)"; return 0; fi
  open_window "$name :$port" "$dir" "$cmd" && ok "$name starting on port $port (new window)"
}

wait_for() {
  local name="$1" port="$2" seconds="$3" i
  [ "$DRY_RUN" = 1 ] && return 0
  for ((i = 0; i < seconds; i++)); do
    port_busy "$port" && { ok "$name answers on port $port"; return 0; }
    sleep 1
  done
  warn "$name is not answering on port $port after ${seconds}s — look at its window"
  return 1
}

check_database() {
  if command -v pg_isready >/dev/null 2>&1; then
    if pg_isready -q -h localhost; then ok "PostgreSQL is running"
    else warn "PostgreSQL is not answering on localhost (start it: sudo systemctl start postgresql)"; fi
  fi
}

do_code() {
  if ! command -v code >/dev/null 2>&1; then warn "VS Code ('code' command) not found"; return 0; fi
  local existing=() p
  for p in "${CODE_PROJECTS[@]}"; do [ -d "$p" ] && existing+=("$p"); done
  if [ "$DRY_RUN" = 1 ]; then
    if [ "$ONE_WINDOW" = 1 ]; then say "  [dry-run] code -n ${existing[*]}"
    else for p in "${existing[@]}"; do say "  [dry-run] code -n $p"; done; fi
    return 0
  fi
  if [ "$ONE_WINDOW" = 1 ]; then
    code -n "${existing[@]}" >/dev/null 2>&1 &
    ok "VS Code: ${#existing[@]} projects in one window"
  else
    # code reuses an open window for a folder it already shows.
    for p in "${existing[@]}"; do code "$p" >/dev/null 2>&1; done
    ok "VS Code: ${#existing[@]} projects opened"
  fi
}

do_back() {
  check_database
  start_one "usersAppAK" "$USERS_DIR" "$USERS_PORT" "$BACKEND_CMD"
  start_one "shao" "$SHAO_DIR" "$SHAO_PORT" "$BACKEND_CMD"
}

do_front() {
  start_one "p3" "$P3_DIR" "$P3_PORT" "PORT=$P3_PORT $FRONTEND_CMD"
  start_one "showcase" "$SHOWCASE_DIR" "$SHOWCASE_PORT" "PORT=$SHOWCASE_PORT $FRONTEND_CMD"
}

do_status() {
  local entry name port
  for entry in "usersAppAK:$USERS_PORT" "shao:$SHAO_PORT" "p3:$P3_PORT" "showcase:$SHOWCASE_PORT"; do
    name=${entry%%:*}; port=${entry##*:}
    if port_busy "$port"; then ok "$name running on port $port"; else warn "$name not running (port $port)"; fi
  done
}

do_stop() {
  local entry name port pids
  for entry in "showcase:$SHOWCASE_PORT" "p3:$P3_PORT" "shao:$SHAO_PORT" "usersAppAK:$USERS_PORT"; do
    name=${entry%%:*}; port=${entry##*:}
    pids=$(port_pids "$port" | tr '\n' ' ')
    if [ -z "${pids// /}" ]; then
      if port_busy "$port"; then warn "$name: port $port busy, but its process is not yours (not stopped)"
      else ok "$name was not running"; fi
      continue
    fi
    if [ "$DRY_RUN" = 1 ]; then say "  [dry-run] kill $pids  ($name)"; continue; fi
    # shellcheck disable=SC2086
    kill $pids 2>/dev/null && ok "$name stopped (port $port)"
  done
}

say "AlephK development — $ACTION"
if [ -z "$TERMINAL" ] && [[ "$ACTION" =~ ^(all|back|front)$ ]]; then
  err "No terminal program found. Install one: sudo apt install gnome-terminal"; exit 1
fi
[ -n "$TERMINAL" ] && [[ "$ACTION" =~ ^(all|back|front)$ ]] && say "  terminal: $TERMINAL"

case "$ACTION" in
  all)
    [ "$NO_CODE" = 1 ] || do_code
    do_back
    say "  waiting for the backends (the first 'go run' compiles, it can take a minute)…"
    wait_for "usersAppAK" "$USERS_PORT" "$BACKEND_WAIT"
    wait_for "shao" "$SHAO_PORT" "$BACKEND_WAIT"
    do_front
    say "Ready: p3 http://localhost:$P3_PORT  ·  store http://localhost:$SHOWCASE_PORT  ·  stop all: $0 stop"
    ;;
  back)   do_back ;;
  front)  do_front ;;
  code)   do_code ;;
  status) do_status ;;
  stop)   do_stop ;;
esac
