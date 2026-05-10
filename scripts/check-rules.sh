#!/bin/bash
# check-rules.sh -- Grep harness for Rules 6/7/8.
# Exits non-zero if any violation is detected.
set -euo pipefail
errors=0

echo "=== Rule Enforcement Checks ==="

# Rule 6: No time.Sleep in tests (use require.Eventually or channel sync).
# Intentional sleeps (simulating delays, real-time waits) are excluded via
# "// intentional:" or "// simulates" or "// requires real-time" comments.
SLEEP_HITS=$(grep -rE 'time\.Sleep\(' --include='*_test.go' . \
    | grep -v 'vendor/' \
    | grep -vE '// (intentional:|simulates |.* requires real-time|poll interval for|required:)' \
    || true)
if [ -n "$SLEEP_HITS" ]; then
    echo "FAIL: Rule 6 - time.Sleep found in test files (use require.Eventually or channel sync)"
    echo "$SLEEP_HITS"
    errors=$((errors + 1))
fi

# Rule 7: No gRPC reflection defaulting to true in server configs.
REFLECTION_HITS=$(grep -rE 'Reflection:\s*true' server/ || true)
if [ -n "$REFLECTION_HITS" ]; then
    echo "FAIL: Rule 7 - Reflection defaults to true in server config"
    echo "$REFLECTION_HITS"
    errors=$((errors + 1))
fi

# Rule 8: No mutable GitHub Actions tags.
if [ -d .github/workflows ]; then
    ACTION_HITS=$(grep -rE 'uses:\s+[^@]+@v[0-9]' .github/workflows/ || true)
    if [ -n "$ACTION_HITS" ]; then
        echo "FAIL: Rule 8 - Mutable action tags found (use SHA pinning)"
        echo "$ACTION_HITS"
        errors=$((errors + 1))
    fi
fi

if [ "$errors" -eq 0 ]; then
    echo "PASS: All rule checks passed"
fi

exit "$errors"
