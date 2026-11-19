#!/usr/bin/env sh

# ---- CONFIG ----------------------------------------------------------------
SESSION=seeker
DIR=${SEEKER_DIR:-$(dirname "$(readlink -f "$0")")}   # where binary + config live
BIN=${SEEKER_BIN:-$DIR/seeker}                        # seeker binary
CONFIG=${SEEKER_CONFIG:-$DIR/config.toml}             # passed as -c
LOGDIR=$DIR/logs
LOGFILE=$LOGDIR/seeker.log
LOGFILE_OLD=$LOGDIR/seeker.old.log
STOP_TIMEOUT=30                             # seconds to wait for clean shutdown
RESTART_DELAY=10                            # seconds between crash respawns
AUTORESTART=1                               # 0 = never respawn on nonzero exit
# -----------------------------------------------------------------------------

# Helpers usable inside the hooks:
#   signal <SIG>   send a signal to the seeker process (e.g. signal INT)
#   is_up          true while the session exists

on_start() {             # runs in $DIR, in the foreground; supervised by the loop
    [ -x "$BIN" ] || { echo "Seeker binary not found/executable: $BIN" >&2; return 1; }
    [ -f "$CONFIG" ] || echo "Config $CONFIG missing, seeker will generate default." >&2
    echo "Starting seeker: $BIN -c $CONFIG"
    "$BIN" -c "$CONFIG"
}

on_stop() {              # seeker handles SIGINT/SIGTERM: closes discord session, exits 0
    signal INT
}

on_restart() {           # default: full cycle
    stop
    sleep 2
    start
}

set -u
SELF=$(readlink -f "$0")
STOPFILE=${TMPDIR:-/tmp}/.$SESSION.stopping
STATUSFILE=${TMPDIR:-/tmp}/.$SESSION.exitcode

is_up()  { tmux has-session -t "$SESSION" 2>/dev/null; }
signal() {
    # seeker runs a few levels below the pane shell; match by exact binary path
    pkill "-${1:-TERM}" -f "^$BIN( |\$)" 2>/dev/null
}
log()    { echo "$*" | tee -a "$LOGFILE"; }   # supervisor message: pane + logfile

_loop() {
    cd "$DIR" || exit 1
    while :; do
        # $? after a pipeline is tee's, not on_start's — stash the real one in
        # a file instead (no PIPESTATUS, this runs under plain sh).
        { on_start; echo $? > "$STATUSFILE"; } 2>&1 | tee -a "$LOGFILE"
        code=$(cat "$STATUSFILE")
        [ -f "$STOPFILE" ]   && { log ">> stop requested, staying down."; break; }
        [ "$code" -eq 0 ]    && { log ">> clean exit, staying down."; break; }
        [ "$AUTORESTART" = 1 ] || { log ">> exit $code, autorestart off."; break; }
        log ">> crashed (exit $code), restarting in ${RESTART_DELAY}s..."
        sleep "$RESTART_DELAY"
    done
}

start() {
    if is_up; then
        echo "Already running — attaching."
        exec tmux attach -t "$SESSION"
    fi
    [ -d "$DIR" ] || { echo "DIR does not exist: $DIR" >&2; exit 1; }
    rm -f "$STOPFILE"
    mkdir -p "$LOGDIR"
    [ -f "$LOGFILE" ] && mv -f "$LOGFILE" "$LOGFILE_OLD"
    tmux new-session -d -s "$SESSION" "$SELF" _loop
    echo "Started '$SESSION'.  attach: $0 attach   stop: $0 stop   logs: $0 logs"
}

stop() {
    is_up || { echo "Not running."; return 0; }
    : > "$STOPFILE"
    echo "Shutting down, waiting up to ${STOP_TIMEOUT}s..."
    on_stop
    i=0
    while is_up; do
        i=$((i + 1))
        [ "$i" -ge "$STOP_TIMEOUT" ] && {
            echo "Timeout — killing session."
            signal KILL
            tmux kill-session -t "$SESSION" 2>/dev/null
            break
        }
        sleep 1
    done
    rm -f "$STOPFILE"
    echo "Stopped."
}

case "${1:-}" in
    _loop)   _loop ;;
    start)   start ;;
    stop)    stop ;;
    restart) on_restart ;;
    attach)  is_up && exec tmux attach -t "$SESSION" || echo "Not running." ;;
    status)  is_up && echo "running" || echo "stopped" ;;
    logs)    [ -f "$LOGFILE" ] && tail -n "${2:-200}" "$LOGFILE" || echo "No log yet: $LOGFILE" ;;
    *) echo "usage: $0 {start|stop|restart|attach|status|logs [n]}"; exit 1 ;;
esac
