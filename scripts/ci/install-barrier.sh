#!/bin/sh
# install-barrier.sh - hold a lane's capture until every route of its adapter
# has installed, across the separate runners a dispatch spreads them over.
#
# ⛔ WHY IT EXISTS. A run manifest says a build was acquired twice before it was
# observed (`E-MAN-03`), and `capture-client` run 37 broke that without anyone
# lying: its release lane observed from 09:36:22 to 09:37:07, on one runner,
# while the source route was still compiling on another until 09:43:40. The
# operator settled on 2026-10-14 that the workflow makes the order true rather
# than the record bending to it. `CI-09`.
#
# ⭐ THE RUNNERS SHARE NOTHING BUT THE COMMIT, so the commit is where they meet.
# Each lane posts one completed check run naming its run, its adapter and its
# route, with conclusion `success` once it installed and `failure` when it could
# not; each then reads the commit's check runs until every route of its adapter
# has arrived. The run attempt is in the name, so a barrier from an earlier
# dispatch of the same commit can never satisfy this one.
#
# ⚠ IT NEVER FAILS A LANE. A route that cannot be satisfied fails its own job
# and must leave the others' measurements intact, which the workflow promises.
# So `wait` always ends 0 once it could run and writes its verdict instead:
# `passed`, `failed` naming the routes that could not install, or `timeout`.
# `assemble-capture` is what refuses a manifest whose order is not true, by the
# timestamps, whatever the verdict says.
#
# Usage:
#   sh scripts/ci/install-barrier.sh arrive <adapter> <route> <success|failure>
#   sh scripts/ci/install-barrier.sh wait <adapter> <routes-json> <record>
#
# Environment: GH_TOKEN, GITHUB_API_URL, GITHUB_REPOSITORY, GITHUB_SHA, and
# BARRIER_RUN, the run's id and attempt. BIT_IDS_BARRIER_TIMEOUT (seconds,
# default 1500) and BIT_IDS_BARRIER_INTERVAL (seconds, default 15) bound the
# wait.
#
# Exit codes: 0 posted, or waited and wrote a verdict; 2 could not run.

set -u

for _need in GH_TOKEN GITHUB_API_URL GITHUB_REPOSITORY GITHUB_SHA BARRIER_RUN; do
  eval "_value=\${$_need:-}"
  [ -n "$_value" ] || {
    printf 'install-barrier: %s is not set\n' "$_need" >&2
    exit 2
  }
done
command -v jq >/dev/null 2>&1 || {
  printf 'install-barrier: jq is not on this host\n' >&2
  exit 2
}

API="$GITHUB_API_URL/repos/$GITHUB_REPOSITORY"
TIMEOUT=${BIT_IDS_BARRIER_TIMEOUT:-1500}
INTERVAL=${BIT_IDS_BARRIER_INTERVAL:-15}

name_of() { # adapter route
  printf 'install-barrier (%s %s-%s)' "$BARRIER_RUN" "$1" "$2"
}

api() { # method url [body-file] -> response on stdout
  if [ $# -ge 3 ]; then
    timeout 20 curl -sS --max-time 15 -X "$1" "$2" \
      -H "Authorization: Bearer $GH_TOKEN" \
      -H 'Accept: application/vnd.github+json' \
      --data-binary "@$3"
  else
    timeout 20 curl -sS --max-time 15 -X "$1" "$2" \
      -H "Authorization: Bearer $GH_TOKEN" \
      -H 'Accept: application/vnd.github+json'
  fi
}

[ $# -ge 1 ] || {
  printf 'install-barrier: a mode is required, arrive or wait\n' >&2
  exit 2
}
MODE=$1
shift

case "$MODE" in
  arrive)
    [ $# -eq 3 ] || {
      printf 'install-barrier: arrive takes <adapter> <route> <success|failure>\n' >&2
      exit 2
    }
    case "$3" in
      success | failure) : ;;
      *)
        printf 'install-barrier: a conclusion is success or failure, not [%s]\n' "$3" >&2
        exit 2
        ;;
    esac
    BODY=$(mktemp) || exit 2
    jq -n --arg name "$(name_of "$1" "$2")" --arg sha "$GITHUB_SHA" --arg c "$3" \
      '{name: $name, head_sha: $sha, status: "completed", conclusion: $c,
        output: {title: ("install " + $c), summary: ("install " + $c)}}' >"$BODY"
    ID=$(api POST "$API/check-runs" "$BODY" | jq -r '.id // empty' 2>/dev/null)
    rm -f "$BODY"
    [ -n "$ID" ] || {
      printf 'install-barrier: the check run for %s %s was not created\n' "$1" "$2" >&2
      exit 2
    }
    printf 'install-barrier: %s arrived, %s\n' "$(name_of "$1" "$2")" "$3"
    exit 0
    ;;

  wait)
    [ $# -eq 3 ] || {
      printf 'install-barrier: wait takes <adapter> <routes-json> <record>\n' >&2
      exit 2
    }
    ADAPTER=$1
    RECORD=$3
    ROUTES=$(printf '%s' "$2" | jq -r '.[]' 2>/dev/null) || ROUTES=""
    [ -n "$ROUTES" ] || {
      printf 'install-barrier: no routes in [%s]\n' "$2" >&2
      exit 2
    }

    START=$(date +%s)
    VERDICT=timeout
    FAILED=""
    while :; do
      ARRIVED=0
      MISSING=""
      FAILED=""
      for _route in $ROUTES; do
        _name=$(name_of "$ADAPTER" "$_route")
        _q=$(printf '%s' "$_name" | jq -sRr @uri)
        _c=$(api GET "$API/commits/$GITHUB_SHA/check-runs?check_name=$_q&per_page=10" |
          jq -r '.check_runs[0].conclusion // empty' 2>/dev/null)
        case "$_c" in
          success) ARRIVED=$((ARRIVED + 1)) ;;
          failure) FAILED="$FAILED $_route" ;;
          *) MISSING="$MISSING $_route" ;;
        esac
      done
      if [ -n "$FAILED" ]; then
        VERDICT=failed
        break
      fi
      if [ -z "$MISSING" ]; then
        VERDICT=passed
        break
      fi
      [ $(($(date +%s) - START)) -lt "$TIMEOUT" ] || break
      sleep "$INTERVAL"
    done

    {
      printf 'bit-ids/install-barrier/1\n'
      printf 'run=%s\n' "$BARRIER_RUN"
      printf 'adapter=%s\n' "$ADAPTER"
      printf 'routes=%s\n' "$(printf '%s' "$ROUTES" | tr '\n' ',' | sed 's/,$//')"
      printf 'verdict=%s\n' "$VERDICT"
      printf 'failed_routes=%s\n' "$(printf '%s' "$FAILED" | sed 's/^ //')"
      printf 'missing_routes=%s\n' "$(printf '%s' "${MISSING:-}" | sed 's/^ //')"
      printf 'waited_seconds=%s\n' "$(($(date +%s) - START))"
      printf 'ended_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    } >"$RECORD" || exit 2
    cat "$RECORD"
    exit 0
    ;;

  *)
    printf 'install-barrier: unknown mode [%s]\n' "$MODE" >&2
    exit 2
    ;;
esac
