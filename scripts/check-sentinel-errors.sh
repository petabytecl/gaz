#!/bin/bash
# check-sentinel-errors.sh -- Rule 5: Sentinel error usage check.
# Verifies that every exported sentinel error in di/errors.go has at least
# one reference in production code (outside its own definition and tests).
set -euo pipefail
errors=0

echo "=== Sentinel Error Usage Checks ==="

# Extract exported sentinel error names from di/errors.go.
# These are var declarations matching Err[A-Z]... = errors.New(...).
SENTINEL_ERRORS=$(grep -oE '\bErr[A-Z][A-Za-z]+\b' di/errors.go | sort -u)

for err in $SENTINEL_ERRORS; do
    count=$(grep -rE "\b${err}\b" --include='*.go' . \
        | grep -v '_test.go' \
        | grep -v 'di/errors.go' \
        | grep -v 'vendor/' \
        | grep -v '^\./linters/' \
        | wc -l)
    if [ "$count" -eq 0 ]; then
        echo "FAIL: Rule 5 - Sentinel error ${err} is exported but never referenced in production code"
        errors=$((errors + 1))
    fi
done

if [ "$errors" -eq 0 ]; then
    echo "PASS: All sentinel errors have production references"
fi

exit "$errors"
