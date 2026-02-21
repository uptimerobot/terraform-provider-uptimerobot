#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

RUNS="${1:-6}"
PKG="${ACC_TEST_PACKAGE:-./internal/provider}"
PATTERN="${ACC_TEST_PATTERN:-^TestAcc}"
TIMEOUT="${ACC_TEST_TIMEOUT:-120m}"
TAGS="${ACC_TEST_TAGS:-acceptance}"
ACC_GO_TEST_FLAGS="${ACC_GO_TEST_FLAGS:--parallel=1 -p=1}"

: "${UPTIMEROBOT_API_KEY:?UPTIMEROBOT_API_KEY must be set}"
: "${UPTIMEROBOT_TEST_ALERT_CONTACT_ID:?UPTIMEROBOT_TEST_ALERT_CONTACT_ID must be set}"

STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${ACC_OUT_DIR:-$ROOT_DIR/.acc-runs/$STAMP}"
mkdir -p "$OUT_DIR"

GOCACHE="${GOCACHE:-$ROOT_DIR/.gocache}"
mkdir -p "$GOCACHE"
export GOCACHE

echo "Acceptance repeat run started"
echo "  runs:      $RUNS"
echo "  package:   $PKG"
echo "  pattern:   $PATTERN"
echo "  timeout:   $TIMEOUT"
echo "  tags:      $TAGS"
echo "  go flags:  $ACC_GO_TEST_FLAGS"
echo "  out dir:   $OUT_DIR"
echo "  gocache:   $GOCACHE"
echo

read -r -a GO_TEST_EXTRA_ARGS <<< "$ACC_GO_TEST_FLAGS"
run_status_file="$OUT_DIR/run-status.tsv"
: > "$run_status_file"

for i in $(seq 1 "$RUNS"); do
  log_file="$OUT_DIR/run${i}.log"
  echo "=== run $i/$RUNS ==="
  go clean -testcache
  set +e
  TF_ACC=1 \
  UPTIMEROBOT_API_KEY="$UPTIMEROBOT_API_KEY" \
  UPTIMEROBOT_TEST_ALERT_CONTACT_ID="$UPTIMEROBOT_TEST_ALERT_CONTACT_ID" \
  GOCACHE="$GOCACHE" \
  go test "$PKG" -tags="$TAGS" -run "$PATTERN" -v -count=1 -timeout="$TIMEOUT" "${GO_TEST_EXTRA_ARGS[@]}" >"$log_file" 2>&1
  rc=$?
  set -e

  tests_ran=0
  if grep -Eq '^=== RUN[[:space:]]+TestAcc' "$log_file"; then
    tests_ran=1
  fi
  if [ "$rc" -eq 0 ] && [ "$tests_ran" -eq 0 ]; then
    rc=2
    {
      echo
      echo "ERROR: No acceptance tests were executed."
      echo "Check ACC_TEST_TAGS (current: '$TAGS') and ACC_TEST_PATTERN (current: '$PATTERN')."
    } >>"$log_file"
  fi
  printf "%s\t%s\t%s\n" "$i" "$rc" "$tests_ran" >> "$run_status_file"

  echo "run=$i exit=$rc"
  tail -n 10 "$log_file" || true
  echo
done

summary_file="$OUT_DIR/summary.md"
fail_counts_file="$OUT_DIR/fail-counts.txt"

cat "$OUT_DIR"/run*.log \
  | sed -nE 's/^--- FAIL: (TestAcc[^ ]+).*/\1/p' \
  | sort \
  | uniq -c \
  | sort -nr > "$fail_counts_file" || true

{
  echo "# Acceptance Repeat Summary"
  echo
  echo "- Date (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "- Runs: $RUNS"
  echo "- Package: \`$PKG\`"
  echo "- Pattern: \`$PATTERN\`"
  echo "- Timeout: \`$TIMEOUT\`"
  echo "- Tags: \`$TAGS\`"
  echo "- Go flags: \`$ACC_GO_TEST_FLAGS\`"
  echo "- Logs: \`$OUT_DIR/runN.log\`"
  echo
  echo "## Per-Run Status"
  for i in $(seq 1 "$RUNS"); do
    log_file="$OUT_DIR/run${i}.log"
    rc="$(awk -v i="$i" '$1 == i { print $2 }' "$run_status_file")"
    tests_ran="$(awk -v i="$i" '$1 == i { print $3 }' "$run_status_file")"
    if [ "$rc" = "0" ]; then
      status="PASS"
    else
      status="FAIL"
    fi
    infra=""
    if grep -q 'timeout waiting on reattach config' "$log_file"; then
      infra=" (provider reattach timeout)"
    fi
    if [ "$tests_ran" = "0" ]; then
      echo "- run $i: $status (no acceptance tests executed)$infra"
    else
      echo "- run $i: $status$infra"
    fi
  done
  echo

  echo "## Stable Failures (Failed In All Runs)"
  stable_failures="$(awk -v runs="$RUNS" '$1 == runs { print $2 }' "$fail_counts_file" || true)"
  if [ -n "$stable_failures" ]; then
    echo "$stable_failures" | sed 's/^/- /'
  else
    echo "- none"
  fi
  echo

  echo "## Flaky Candidates (Failed In Some Runs)"
  flaky_failures="$(awk -v runs="$RUNS" '$1 > 0 && $1 < runs { printf "%s (%s/%s)\n", $2, $1, runs }' "$fail_counts_file" || true)"
  if [ -n "$flaky_failures" ]; then
    echo "$flaky_failures" | sed 's/^/- /'
  else
    echo "- none"
  fi
  echo

  echo "## Notes"
  if grep -q 'timeout waiting on reattach config' "$OUT_DIR"/run*.log; then
    echo "- Infrastructure blocker detected: provider process reattach timeout occurred."
    echo "- Resolve this first before classifying flaky acceptance tests."
  else
    echo "- No provider reattach timeout signatures detected."
  fi
} > "$summary_file"

echo "Summary written to: $summary_file"
