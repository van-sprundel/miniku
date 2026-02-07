#!/bin/bash
set -e

# packages with actual logic (exclude cmd/, testutil, types)
PKGS="./pkg/api/... ./pkg/client/... ./pkg/scheduler/... ./pkg/controller/... ./pkg/kubelet/... ./pkg/store/..."

go test -coverprofile=coverage.out $PKGS > /dev/null 2>&1

TOTAL=$(go tool cover -func=coverage.out | grep ^total | awk '{print $3}')
echo "Coverage: $TOTAL"

# optional: open html report
if [ "$1" = "--html" ]; then
    go tool cover -html=coverage.out
fi
